package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"events-audit/internal/constants"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

// Config holds NATS JetStream configuration.
type Config struct {
	URL             string
	Subject         string
	StreamName      string
	ConsumerName    string
	DurableName     string
	Timeout         time.Duration
	MaxDeliver      int
	AckWait         time.Duration
	DeliverPolicy   int
	ReplayPolicy    int
	PullMaxMessages int
	PullTimeout     time.Duration
	CreateStream    bool
	StreamMaxAge    time.Duration
	StreamMaxBytes  int64
	StreamMaxMsgs   int64
	StreamReplicas  int
}

// DefaultConfig returns default JetStream configuration.
func DefaultConfig() Config {
	return Config{
		URL:             "nats://localhost:4222",
		Subject:         "events.>",
		StreamName:      "EVENTS",
		ConsumerName:    "events-audit-consumer",
		DurableName:     "events-audit-durable",
		Timeout:         constants.DefaultTimeout,
		MaxDeliver:      constants.DefaultMaxDeliver,
		AckWait:         constants.DefaultAckWait,
		DeliverPolicy:   0, // DeliverAllPolicy
		ReplayPolicy:    0, // ReplayInstantPolicy
		PullMaxMessages: constants.DefaultPullMaxMessages,
		PullTimeout:     constants.DefaultPullTimeout,
		CreateStream:    true,
		StreamMaxAge:    constants.DefaultStreamMaxAge,
		StreamMaxBytes:  constants.DefaultStreamMaxBytes,
		StreamMaxMsgs:   constants.DefaultStreamMaxMsgs,
		StreamReplicas:  constants.DefaultStreamReplicas,
	}
}

// Client wraps NATS JetStream connection and provides event handling.
type Client struct {
	conn         *nats.Conn
	js           nats.JetStreamContext
	config       Config
	logger       *logrus.Logger
	subscription *nats.Subscription
	consumer     nats.ConsumerInfo
}

// EventHandler defines the function signature for handling JetStream events.
type EventHandler func(msg *nats.Msg) error

// NewClient creates a new NATS JetStream client.
func NewClient(config Config, logger *logrus.Logger) (*Client, error) {
	if logger == nil {
		logger = logrus.New()
	}

	client := &Client{
		config: config,
		logger: logger,
	}

	return client, nil
}

// Connect establishes connection to NATS server and initializes JetStream.
func (c *Client) Connect(_ context.Context) error {
	opts := []nats.Option{
		nats.Name("events-audit-jetstream-listener"),
		nats.Timeout(c.config.Timeout),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				c.logger.WithError(err).Error("NATS disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			c.logger.WithField("url", nc.ConnectedUrl()).Info("NATS reconnected")
			// Reinitialize JetStream context after reconnect
			if js, jsErr := nc.JetStream(); jsErr != nil {
				c.logger.WithError(jsErr).Error("Failed to reinitialize JetStream after reconnect")
			} else {
				c.js = js
				c.logger.Info("JetStream context reinitialized after reconnect")
			}
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			c.logger.Info("NATS connection closed")
		}),
		nats.ErrorHandler(func(_ *nats.Conn, sub *nats.Subscription, err error) {
			c.logger.WithError(err).WithField("subject", sub.Subject).Error("NATS error")
		}),
	}

	conn, err := nats.Connect(c.config.URL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	c.conn = conn
	c.logger.WithField("url", c.config.URL).Info("Connected to NATS")

	// Initialize JetStream context
	js, err := conn.JetStream()
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	c.js = js
	c.logger.Info("JetStream context initialized")

	// Create or update stream if required
	if c.config.CreateStream {
		if streamErr := c.ensureStream(); streamErr != nil {
			c.conn.Close()
			return fmt.Errorf("failed to ensure stream: %w", streamErr)
		}
	}

	return nil
}

// ensureStream creates or updates the JetStream stream.
func (c *Client) ensureStream() error {
	streamConfig := &nats.StreamConfig{
		Name:      c.config.StreamName,
		Subjects:  []string{c.config.Subject},
		MaxAge:    c.config.StreamMaxAge,
		MaxBytes:  c.config.StreamMaxBytes,
		MaxMsgs:   c.config.StreamMaxMsgs,
		Replicas:  c.config.StreamReplicas,
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		Discard:   nats.DiscardOld,
	}

	streamInfo, err := c.getOrCreateStream(streamConfig)
	if err != nil {
		return err
	}

	c.logStreamReady(streamInfo)
	return nil
}

// getOrCreateStream gets existing stream or creates a new one.
func (c *Client) getOrCreateStream(streamConfig *nats.StreamConfig) (*nats.StreamInfo, error) {
	streamInfo, err := c.js.StreamInfo(c.config.StreamName)
	if err != nil {
		return c.handleStreamNotFound(err, streamConfig)
	}

	return c.handleExistingStream(streamInfo, streamConfig)
}

// handleStreamNotFound creates a new stream if it doesn't exist.
func (c *Client) handleStreamNotFound(err error, streamConfig *nats.StreamConfig) (*nats.StreamInfo, error) {
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	streamInfo, createErr := c.js.AddStream(streamConfig)
	if createErr != nil {
		return nil, fmt.Errorf("failed to create stream %s: %w", c.config.StreamName, createErr)
	}

	c.logger.WithFields(logrus.Fields{
		"stream":   c.config.StreamName,
		"subjects": streamConfig.Subjects,
	}).Info("Created JetStream stream")

	return streamInfo, nil
}

// handleExistingStream updates stream if necessary.
func (c *Client) handleExistingStream(
	streamInfo *nats.StreamInfo,
	streamConfig *nats.StreamConfig,
) (*nats.StreamInfo, error) {
	if c.needsStreamUpdate(streamInfo) {
		return c.updateStream(streamConfig)
	}

	c.logger.WithField("stream", c.config.StreamName).Info("JetStream stream already exists and is up to date")
	return streamInfo, nil
}

// needsStreamUpdate checks if stream configuration needs updating.
func (c *Client) needsStreamUpdate(streamInfo *nats.StreamInfo) bool {
	return streamInfo.Config.MaxAge != c.config.StreamMaxAge ||
		streamInfo.Config.MaxBytes != c.config.StreamMaxBytes ||
		streamInfo.Config.MaxMsgs != c.config.StreamMaxMsgs
}

// updateStream updates the stream configuration.
func (c *Client) updateStream(streamConfig *nats.StreamConfig) (*nats.StreamInfo, error) {
	streamInfo, err := c.js.UpdateStream(streamConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to update stream %s: %w", c.config.StreamName, err)
	}

	c.logger.WithField("stream", c.config.StreamName).Info("Updated JetStream stream")
	return streamInfo, nil
}

// logStreamReady logs stream readiness information.
func (c *Client) logStreamReady(streamInfo *nats.StreamInfo) {
	c.logger.WithFields(logrus.Fields{
		"stream":    streamInfo.Config.Name,
		"messages":  streamInfo.State.Msgs,
		"bytes":     streamInfo.State.Bytes,
		"consumers": streamInfo.State.Consumers,
	}).Info("JetStream stream ready")
}

// ensureConsumer creates or recreates consumer with proper configuration.
func (c *Client) ensureConsumer(consumerConfig *nats.ConsumerConfig) (*nats.ConsumerInfo, error) {
	// Try to get existing consumer
	existingConsumer, err := c.js.ConsumerInfo(c.config.StreamName, c.config.DurableName)
	if err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}

	// If consumer exists, check if it's configured correctly for pull-based subscription
	if existingConsumer != nil {
		if existingConsumer.Config.DeliverSubject != "" {
			// Consumer has DeliverSubject, it's push-based, need to delete and recreate
			c.logger.WithField("consumer", c.config.DurableName).Warn("Found push-based consumer, deleting to recreate as pull-based")

			if deleteErr := c.js.DeleteConsumer(c.config.StreamName, c.config.DurableName); deleteErr != nil {
				return nil, fmt.Errorf("failed to delete existing push-based consumer: %w", deleteErr)
			}

			c.logger.WithField("consumer", c.config.DurableName).Info("Deleted existing push-based consumer")
		} else {
			// Consumer is already pull-based, can reuse it
			c.logger.WithField("consumer", c.config.DurableName).Info("Found existing pull-based consumer, reusing")
			return existingConsumer, nil
		}
	}

	// Create new pull-based consumer
	consumerInfo, err := c.js.AddConsumer(c.config.StreamName, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull-based consumer: %w", err)
	}

	c.logger.WithField("consumer", c.config.DurableName).Info("Created new pull-based consumer")
	return consumerInfo, nil
}

// Subscribe creates a durable consumer and subscribes to messages.
func (c *Client) Subscribe(ctx context.Context, handler EventHandler) error {
	if c.js == nil {
		return errors.New("JetStream context not initialized")
	}

	// Consumer configuration for pull-based subscription
	consumerConfig := &nats.ConsumerConfig{
		Name:          c.config.ConsumerName,
		Durable:       c.config.DurableName,
		DeliverPolicy: nats.DeliverPolicy(c.config.DeliverPolicy),
		ReplayPolicy:  nats.ReplayPolicy(c.config.ReplayPolicy),
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.config.AckWait,
		MaxDeliver:    c.config.MaxDeliver,
		FilterSubject: c.config.Subject,
		// DeliverSubject removed for pull-based subscription
	}

	// Create or get consumer
	consumerInfo, err := c.ensureConsumer(consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to ensure consumer: %w", err)
	}

	c.consumer = *consumerInfo
	c.logger.WithFields(logrus.Fields{
		"consumer":    c.config.ConsumerName,
		"stream":      c.config.StreamName,
		"durable":     c.config.DurableName,
		"filter":      c.config.Subject,
		"ack_pending": consumerInfo.NumAckPending,
		"delivered":   consumerInfo.Delivered.Consumer,
	}).Info("Created JetStream consumer")

	// Subscribe using pull-based subscription
	sub, err := c.js.PullSubscribe(c.config.Subject, c.config.DurableName)
	if err != nil {
		return fmt.Errorf("failed to create pull subscription: %w", err)
	}

	c.subscription = sub
	c.logger.WithField("subject", c.config.Subject).Info("Subscribed to JetStream")

	// Start message processing loop
	return c.processMessages(ctx, handler)
}

// processMessages handles the pull-based message processing loop.
func (c *Client) processMessages(ctx context.Context, handler EventHandler) error {
	c.logger.Info("Starting JetStream message processing loop")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping message processing")
			return ctx.Err()
		default:
			// Fetch messages
			msgs, err := c.subscription.Fetch(c.config.PullMaxMessages, nats.MaxWait(c.config.PullTimeout))
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) {
					// Timeout is expected when no messages are available
					continue
				}
				c.logger.WithError(err).Error("Failed to fetch messages")
				// Don't return error, just continue trying
				time.Sleep(time.Second)
				continue
			}

			// Process each message
			for _, msg := range msgs {
				if processErr := c.processMessage(msg, handler); processErr != nil {
					c.logger.WithError(processErr).WithFields(logrus.Fields{
						"subject": msg.Subject,
						"stream":  msg.Reply,
					}).Error("Failed to process message")
				}
			}
		}
	}
}

// processMessage handles individual message processing with acknowledgment.
func (c *Client) processMessage(msg *nats.Msg, handler EventHandler) error {
	startTime := time.Now()

	// Get message metadata
	meta, err := msg.Metadata()
	if err != nil {
		c.logger.WithError(err).Error("Failed to get message metadata")
		// Nak the message so it can be redelivered
		return msg.Nak()
	}

	c.logger.WithFields(logrus.Fields{
		"subject":   msg.Subject,
		"stream":    meta.Stream,
		"consumer":  meta.Consumer,
		"delivered": meta.NumDelivered,
		"pending":   meta.NumPending,
		"timestamp": meta.Timestamp.Format(time.RFC3339),
		"size":      len(msg.Data),
	}).Debug("Processing JetStream message")

	// Call the handler
	if handlerErr := handler(msg); handlerErr != nil {
		// Check if this message has been delivered too many times
		if meta.NumDelivered >= uint64(c.config.MaxDeliver) { //nolint:gosec // safe conversion from int to uint64
			c.logger.WithError(handlerErr).WithFields(logrus.Fields{
				"subject":     msg.Subject,
				"delivered":   meta.NumDelivered,
				"max_deliver": c.config.MaxDeliver,
			}).Error("Message exceeded max delivery attempts, sending terminal acknowledgment")
			return msg.Term()
		}

		c.logger.WithError(handlerErr).WithFields(logrus.Fields{
			"subject":   msg.Subject,
			"delivered": meta.NumDelivered,
		}).Error("Handler failed, negative acknowledging message")
		return msg.Nak()
	}

	// Acknowledge successful processing
	if ackErr := msg.Ack(); ackErr != nil {
		c.logger.WithError(ackErr).Error("Failed to acknowledge message")
		return ackErr
	}

	processingTime := time.Since(startTime)
	c.logger.WithFields(logrus.Fields{
		"subject":         msg.Subject,
		"processing_time": processingTime.String(),
		"delivered":       meta.NumDelivered,
	}).Debug("Message processed and acknowledged")

	return nil
}

// GetStreamInfo returns information about the stream.
func (c *Client) GetStreamInfo() (*nats.StreamInfo, error) {
	if c.js == nil {
		return nil, errors.New("JetStream context not initialized")
	}
	return c.js.StreamInfo(c.config.StreamName)
}

// GetConsumerInfo returns information about the consumer.
func (c *Client) GetConsumerInfo() (*nats.ConsumerInfo, error) {
	if c.js == nil {
		return nil, errors.New("JetStream context not initialized")
	}
	return c.js.ConsumerInfo(c.config.StreamName, c.config.DurableName)
}

// Close closes the subscription and NATS connection.
func (c *Client) Close() error {
	c.logger.Info("Closing JetStream client")

	if c.subscription != nil {
		if err := c.subscription.Unsubscribe(); err != nil {
			c.logger.WithError(err).Error("Failed to unsubscribe")
		}
	}

	if c.conn != nil {
		c.conn.Close()
		c.logger.Info("NATS connection closed")
	}

	return nil
}

// IsConnected returns true if connected to NATS.
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Drain gracefully drains the subscription.
func (c *Client) Drain() error {
	if c.subscription != nil {
		return c.subscription.Drain()
	}
	return nil
}
