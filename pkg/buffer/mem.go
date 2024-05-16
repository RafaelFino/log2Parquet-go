package buffer

import (
	"bytes"
	"context"
	"data2parquet/pkg/config"
	"data2parquet/pkg/domain"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// / Mem buffer
// / @struct Mem
// / @implements Buffer
type Mem struct {
	config   *config.Config
	data     map[string][]domain.Record
	recovery map[string][]domain.Record
	dlq      []*DLQData
	buff     chan BuffItem
	mu       sync.Mutex
	Ready    bool
	ctx      context.Context
}

type BuffItem struct {
	key  string
	item domain.Record
}

// / New mem buffer
// / @param config *config.Config
// / @return Buffer
func NewMem(ctx context.Context, config *config.Config) Buffer {
	ret := &Mem{
		data:     make(map[string][]domain.Record),
		recovery: make(map[string][]domain.Record),
		config:   config,
		buff:     make(chan BuffItem, config.BufferSize),
		ctx:      ctx,
		dlq:      make([]*DLQData, 0),
	}

	ret.buff = make(chan BuffItem, config.BufferSize)
	signal := make(chan bool)

	go func(m *Mem, signal chan bool) {
		signal <- true
		for item := range m.buff {
			m.mu.Lock()
			if _, ok := m.data[item.key]; !ok {
				m.data[item.key] = make([]domain.Record, 0, m.config.BufferSize)
			}

			m.data[item.key] = append(m.data[item.key], item.item)
			m.mu.Unlock()
		}
	}(ret, signal)

	ret.Ready = <-signal

	return ret
}

func (m *Mem) Close() error {
	slog.Debug("Closing buffer", "module", "buffer.mem", "function", "Close")
	m.Ready = false
	close(m.buff)
	return nil
}

func (m *Mem) Len(key string) int {
	slog.Debug("Getting buffer length", "key", key, "module", "buffer.mem", "function", "Len")
	if m.data == nil {
		return 0
	}

	if _, ok := m.data[key]; !ok {
		return 0
	}

	return len(m.data[key])
}

func (m *Mem) Push(key string, item domain.Record) error {
	if item == nil {
		slog.Warn("Item is nil	", "key", key, "module", "buffer.mem", "function", "Push")
		return errors.New("item is nil")
	}

	m.buff <- BuffItem{
		key:  key,
		item: item,
	}

	return nil
}

func (m *Mem) PushRecovery(key string, item domain.Record) error {
	if item == nil {
		slog.Warn("Item is nil	", "key", key, "module", "buffer.mem", "function", "Push")
		return errors.New("item is nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	slog.Debug("Pushing to recovery", "key", key, "record", item, "module", "buffer.mem", "function", "PushRecovery")

	if _, ok := m.recovery[key]; !ok {
		m.recovery[key] = make([]domain.Record, 0, m.config.BufferSize)
	}

	m.recovery[key] = append(m.recovery[key], item)

	return nil
}

func (m *Mem) RecoveryData() error {
	slog.Debug("Recovering data", "module", "buffer.mem", "function", "RecoveryData")
	if m.recovery == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for key, records := range m.recovery {
		if _, ok := m.data[key]; !ok {
			m.data[key] = make([]domain.Record, 0, m.config.BufferSize)
		}

		m.data[key] = append(m.data[key], records...)
		m.recovery[key] = make([]domain.Record, 0)
	}

	return nil
}

func (m *Mem) Get(key string) []domain.Record {
	slog.Debug("Getting buffer", "key", key, "module", "buffer.mem", "function", "Get")

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.data == nil {
		return nil
	}

	if _, ok := m.data[key]; !ok {
		return nil
	}

	return m.data[key]
}

func (m *Mem) Clear(key string, size int) error {
	slog.Debug("Clearing buffer", "key", key, "size", size, "module", "buffer.mem", "function", "Clear")
	if m.data == nil {
		return nil
	}

	if _, ok := m.data[key]; !ok {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if size == -1 || size > len(m.data[key]) {
		delete(m.data, key)
		return nil
	}

	m.data[key] = m.data[key][size:]

	return nil
}
func (m *Mem) Keys() []string {
	keys := make([]string, 0, len(m.data))

	for k := range m.data {
		keys = append(keys, k)
	}

	return keys
}

func (m *Mem) IsReady() bool {
	return m.Ready
}

func (m *Mem) HasRecovery() bool {
	return len(m.recovery) > 0
}

func (m *Mem) PushDLQ(key string, buf *bytes.Buffer) error {
	slog.Debug("Pushing to DLQ", "key", key, "module", "buffer.mem", "function", "PushDLQ")

	m.mu.Lock()
	defer m.mu.Unlock()

	m.dlq = append(m.dlq, &DLQData{
		Key:       key,
		Data:      buf.Bytes(),
		Timestamp: time.Now(),
	})

	return nil
}

func (m *Mem) GetDLQ() []*DLQData {
	slog.Debug("Getting DLQ", "module", "buffer.mem", "function", "GetDLQ")

	return m.dlq
}

func (m *Mem) ClearDLQ() error {
	slog.Debug("Clearing DLQ", "module", "buffer.mem", "function", "ClearDLQ")

	m.mu.Lock()
	defer m.mu.Unlock()

	m.dlq = make([]*DLQData, 0)

	return nil
}
