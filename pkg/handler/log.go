package handler

import (
	"data2parquet/pkg/config"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
}

func NewLogHandler(config *config.Config) *LogHandler {
	return &LogHandler{}
}

func (h *LogHandler) Write(ctx *gin.Context) {
	start := time.Now()

	slog.Debug("Write record", "module", "handler", "function", "Write")

	ctx.JSON(http.StatusCreated, gin.H{
		"timestamp": time.Now().Unix(),
		"elapsed":   time.Since(start).String(),
	})
}

func (h *LogHandler) Healthcheck(ctx *gin.Context) {
	start := time.Now()

	slog.Debug("Healthcheck", "module", "handler", "function", "Healthcheck")

	ctx.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"elapsed":   time.Since(start).String(),
	})
}
