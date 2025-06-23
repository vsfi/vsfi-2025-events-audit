package server

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"events-audit/internal/constants"
	"events-audit/internal/nats"

	"github.com/sirupsen/logrus"
)

// Config holds server configuration.
type Config struct {
	NatsURL         string
	NatsSubject     string
	StreamName      string
	ConsumerName    string
	DurableName     string
	CreateStream    bool
	MaxDeliver      int
	AckWait         time.Duration
	PullMaxMessages int
	PullTimeout     time.Duration
	StreamMaxAge    time.Duration
	StreamMaxBytes  int64
	StreamMaxMsgs   int64
	StreamReplicas  int
	LogLevel        string
	LogFormat       string
}

// Server represents the main server.
type Server struct {
	config      Config
	logger      *logrus.Logger
	natsClient  *nats.Client
	eventLogger *nats.EventLogger
}

// NewServer creates a new server instance.
func NewServer(config Config) *Server {
	logger := logrus.New()

	// Configure logger format
	if config.LogFormat == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "02.01.2006 15:04:05.000",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "02.01.2006 15:04:05.000",
			FullTimestamp:   true,
		})
	}

	// Set log level
	if level, err := logrus.ParseLevel(config.LogLevel); err == nil {
		logger.SetLevel(level)
	} else {
		logger.SetLevel(logrus.InfoLevel)
		logger.WithError(err).Warn("Invalid log level, using INFO")
	}

	// Set default values if not provided
	if config.StreamName == "" {
		config.StreamName = "EVENTS"
	}
	if config.ConsumerName == "" {
		config.ConsumerName = "events-audit-consumer"
	}
	if config.DurableName == "" {
		config.DurableName = "events-audit-durable"
	}
	if config.MaxDeliver == 0 {
		config.MaxDeliver = constants.DefaultMaxDeliver
	}
	if config.AckWait == 0 {
		config.AckWait = constants.DefaultAckWait
	}
	if config.PullMaxMessages == 0 {
		config.PullMaxMessages = constants.DefaultPullMaxMessages
	}
	if config.PullTimeout == 0 {
		config.PullTimeout = constants.DefaultPullTimeout
	}
	if config.StreamMaxAge == 0 {
		config.StreamMaxAge = constants.DefaultStreamMaxAge
	}
	if config.StreamMaxBytes == 0 {
		config.StreamMaxBytes = constants.DefaultStreamMaxBytes
	}
	if config.StreamMaxMsgs == 0 {
		config.StreamMaxMsgs = constants.DefaultStreamMaxMsgs
	}
	if config.StreamReplicas == 0 {
		config.StreamReplicas = constants.DefaultStreamReplicas
	}

	return &Server{
		config:      config,
		logger:      logger,
		eventLogger: nats.NewEventLogger(logger),
	}
}

// Run starts the server.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("Starting JetStream events audit server")

	// Create NATS JetStream client
	natsConfig := nats.Config{
		URL:             s.config.NatsURL,
		Subject:         s.config.NatsSubject,
		StreamName:      s.config.StreamName,
		ConsumerName:    s.config.ConsumerName,
		DurableName:     s.config.DurableName,
		Timeout:         constants.DefaultTimeout,
		MaxDeliver:      s.config.MaxDeliver,
		AckWait:         s.config.AckWait,
		DeliverPolicy:   0, // DeliverAllPolicy
		ReplayPolicy:    0, // ReplayInstantPolicy
		PullMaxMessages: s.config.PullMaxMessages,
		PullTimeout:     s.config.PullTimeout,
		CreateStream:    s.config.CreateStream,
		StreamMaxAge:    s.config.StreamMaxAge,
		StreamMaxBytes:  s.config.StreamMaxBytes,
		StreamMaxMsgs:   s.config.StreamMaxMsgs,
		StreamReplicas:  s.config.StreamReplicas,
	}

	var err error
	s.natsClient, err = nats.NewClient(natsConfig, s.logger)
	if err != nil {
		return err
	}

	// Connect to NATS and initialize JetStream
	if connectErr := s.natsClient.Connect(ctx); connectErr != nil {
		return connectErr
	}
	defer s.natsClient.Close()

	// Log stream and consumer information
	if streamInfo, streamErr := s.natsClient.GetStreamInfo(); streamErr == nil {
		s.logger.WithFields(logrus.Fields{
			"stream":    streamInfo.Config.Name,
			"subjects":  streamInfo.Config.Subjects,
			"messages":  streamInfo.State.Msgs,
			"bytes":     streamInfo.State.Bytes,
			"consumers": streamInfo.State.Consumers,
		}).Info("JetStream stream information")
	}

	if consumerInfo, consumerErr := s.natsClient.GetConsumerInfo(); consumerErr == nil {
		s.logger.WithFields(logrus.Fields{
			"consumer":    consumerInfo.Config.Name,
			"durable":     consumerInfo.Config.Durable,
			"delivered":   consumerInfo.Delivered.Consumer,
			"ack_pending": consumerInfo.NumAckPending,
			"redelivered": consumerInfo.NumRedelivered,
		}).Info("JetStream consumer information")
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		s.logger.Info("Received shutdown signal, initiating graceful shutdown")
		// Drain subscription before canceling context
		if drainErr := s.natsClient.Drain(); drainErr != nil {
			s.logger.WithError(drainErr).Error("Failed to drain subscription")
		}
		cancel()
	}()

	// Start subscribing to JetStream events
	s.logger.WithFields(logrus.Fields{
		"subject":  s.config.NatsSubject,
		"stream":   s.config.StreamName,
		"consumer": s.config.ConsumerName,
		"durable":  s.config.DurableName,
	}).Info("Starting to listen for JetStream events")

	err = s.natsClient.Subscribe(ctx, s.eventLogger.HandleEvent)
	if err != nil && !errors.Is(err, context.Canceled) {
		s.logger.WithError(err).Error("Error during JetStream subscription")
		return err
	}

	s.logger.Info("JetStream events audit server stopped")
	return nil
}

// Stop gracefully stops the server.
func (s *Server) Stop() error {
	s.logger.Info("Stopping JetStream server")
	if s.natsClient != nil {
		// Drain first, then close
		if err := s.natsClient.Drain(); err != nil {
			s.logger.WithError(err).Error("Failed to drain subscription during stop")
		}
		return s.natsClient.Close()
	}
	return nil
}
