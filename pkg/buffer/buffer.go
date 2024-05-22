package buffer

import (
	"bytes"
	"context"
	"data2parquet/pkg/config"
	"data2parquet/pkg/domain"
	"log/slog"
	"time"

	msgp "github.com/vmihailenco/msgpack/v5"
)

type Buffer interface {
	Close() error
	Push(key string, item domain.Record) (int, error)
	PushDLQ(key string, item domain.Record) error
	GetDLQ() (map[string][]domain.Record, error)
	ClearDLQ() error
	Get(key string) []domain.Record
	Clear(key string, size int) error
	Len(key string) int
	Keys() []string
	IsReady() bool
	HasRecovery() bool
	PushRecovery(key string, buf *bytes.Buffer) error
	GetRecovery() ([]*RecoveryData, error)
	ClearRecoveryData() error
	CheckLock(key string) bool
}

func New(ctx context.Context, cfg *config.Config) Buffer {
	switch cfg.BufferType {
	case config.BufferTypeRedis:
		return NewRedis(ctx, cfg)
	case config.BufferTypeMem:
		return NewMem(ctx, cfg)
	default:
		return NewMem(ctx, cfg)
	}
}

type RecoveryData struct {
	Key       string    `msg:"key"`
	Data      []byte    `msg:"data"`
	Timestamp time.Time `msg:"timestamp"`
}

func (l *RecoveryData) ToMsgPack() []byte {
	data, err := msgp.Marshal(l)

	if err != nil {
		slog.Error("Error marshalling MsgPack", "error", err)
		return nil
	}

	return data
}

func (l *RecoveryData) FromMsgPack(data []byte) error {
	err := msgp.Unmarshal(data, l)

	if err != nil {
		slog.Error("Error unmarshalling MsgPack", "error", err)
		return err
	}

	return nil
}
