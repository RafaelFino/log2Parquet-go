package server

import (
	"data2parquet/pkg/config"
	"data2parquet/pkg/handler"
	"data2parquet/pkg/receiver"
	"fmt"
	"log"
	"log/slog"

	"net/http"

	"github.com/gin-gonic/gin"
)

type Server struct {
	engine *gin.Engine
	srv    *http.Server

	config   *config.Config
	handler  *handler.LogHandler
	receiver *receiver.Receiver
}

func NewServer(config *config.Config) *Server {
	s := &Server{
		engine:   gin.Default(),
		config:   config,
		receiver: receiver.NewReceiver(config),
	}

	slog.Debug("Starting server", "config", config.ToString(), "module", "server", "function", "NewServer")

	s.handler = handler.NewLogHandler(config)

	gin.ForceConsoleColor()
	gin.DefaultWriter = log.Writer()
	gin.DefaultErrorWriter = log.Writer()

	if s.config.Debug {
		slog.Debug("Debug mode enabled", "module", "server", "function", "NewServer")
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s.engine = gin.Default()
	s.engine.POST("/log/", s.handler.Write)
	s.engine.POST("/healthcheck/", s.handler.Healthcheck)

	s.srv = &http.Server{
		Addr:    s.makeAddress(),
		Handler: s.engine,
	}

	return s
}

func (s *Server) Run() {
	slog.Debug("Starting server", "address", s.makeAddress(), "module", "server", "function", "Run")
	err := s.srv.ListenAndServe()
	if err != nil {
		slog.Debug("Error starting server: %s", err, "module", "server", "function", "Run")
		panic(err)
	}
}

func (s *Server) Stop() error {
	slog.Debug("[Stopping receiver", "module", "server", "function", "Stop")
	err := s.receiver.Close()

	if err != nil {
		slog.Debug("Error stopping service", "error", err, "module", "server", "function", "Stop")
	}

	err = s.srv.Close()

	if err != nil {
		slog.Debug("Error stopping server", "error", err, "module", "server", "function", "Stop")
	}

	return err
}

func (s *Server) makeAddress() string {
	return fmt.Sprintf("%s:%d", s.config.Address, s.config.Port)
}
