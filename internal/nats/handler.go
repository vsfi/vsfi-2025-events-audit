package nats

import (
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

// Event represents a generic event structure.
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// EventLogger handles and logs events from NATS JetStream.
type EventLogger struct {
	logger *logrus.Logger
}

// NewEventLogger creates a new event logger.
func NewEventLogger(logger *logrus.Logger) *EventLogger {
	if logger == nil {
		logger = logrus.New()
	}

	return &EventLogger{
		logger: logger,
	}
}

// HandleEvent processes incoming JetStream messages and logs them.
func (el *EventLogger) HandleEvent(msg *nats.Msg) error {
	// Base fields for all log entries
	baseFields := logrus.Fields{
		"subject":   msg.Subject,
		"data_size": len(msg.Data),
		"reply":     msg.Reply,
	}

	// Try to get JetStream metadata (optional)
	meta, err := msg.Metadata()
	if err == nil && meta != nil {
		// Add JetStream metadata if available
		baseFields["stream"] = meta.Stream
		baseFields["consumer"] = meta.Consumer
		baseFields["sequence"] = meta.Sequence.Stream
		baseFields["delivered"] = meta.NumDelivered
		baseFields["pending"] = meta.NumPending
		baseFields["js_timestamp"] = meta.Timestamp.Format(time.RFC3339)
	}

	// Try to parse as JSON event
	var event Event
	if parseErr := json.Unmarshal(msg.Data, &event); parseErr != nil {
		// If not JSON, log as raw message
		baseFields["raw_data"] = string(msg.Data)
		el.logger.WithFields(baseFields).Info("Received raw event")
		return nil //nolint:nilerr // raw messages are not errors, just log them
	}

	// Log structured event with additional event fields
	eventFields := make(logrus.Fields)
	for k, v := range baseFields {
		eventFields[k] = v
	}
	eventFields["event_id"] = event.ID
	eventFields["event_type"] = event.Type
	eventFields["source"] = event.Source
	eventFields["event_timestamp"] = event.Timestamp.Format(time.RFC3339)
	eventFields["event_data"] = event.Data

	el.logger.WithFields(eventFields).Info("Received structured event")

	return nil
}

// HandleRawEvent processes raw messages without JSON parsing.
func (el *EventLogger) HandleRawEvent(msg *nats.Msg) error {
	fields := logrus.Fields{
		"subject":   msg.Subject,
		"data":      string(msg.Data),
		"data_size": len(msg.Data),
		"reply":     msg.Reply,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Try to get JetStream metadata (optional)
	meta, err := msg.Metadata()
	if err == nil && meta != nil {
		// Add JetStream metadata if available
		fields["stream"] = meta.Stream
		fields["consumer"] = meta.Consumer
		fields["sequence"] = meta.Sequence.Stream
		fields["delivered"] = meta.NumDelivered
		fields["pending"] = meta.NumPending
		fields["js_timestamp"] = meta.Timestamp.Format(time.RFC3339)
	}

	el.logger.WithFields(fields).Info("Received raw event")

	return nil
}

// HandleEventWithCustomFields allows custom field extraction from messages.
func (el *EventLogger) HandleEventWithCustomFields(msg *nats.Msg) error {
	fields := logrus.Fields{
		"subject":   msg.Subject,
		"data_size": len(msg.Data),
		"reply":     msg.Reply,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Try to get JetStream metadata (optional)
	meta, err := msg.Metadata()
	if err == nil && meta != nil {
		// Add JetStream metadata if available
		fields["stream"] = meta.Stream
		fields["consumer"] = meta.Consumer
		fields["sequence"] = meta.Sequence.Stream
		fields["delivered"] = meta.NumDelivered
		fields["pending"] = meta.NumPending
		fields["js_timestamp"] = meta.Timestamp.Format(time.RFC3339)
	}

	// Try to extract additional fields from JSON
	var jsonData map[string]interface{}
	if parseErr := json.Unmarshal(msg.Data, &jsonData); parseErr == nil {
		// Add JSON fields to log
		for k, v := range jsonData {
			fields[k] = v
		}
	} else {
		// If not JSON, add raw data
		fields["raw_data"] = string(msg.Data)
	}

	el.logger.WithFields(fields).Info("Received event with custom fields")
	return nil
}
