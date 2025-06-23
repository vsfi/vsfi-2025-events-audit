package nats_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"events-audit/internal/nats"

	natsclient "github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	natscontainer "github.com/testcontainers/testcontainers-go/modules/nats"
)

func setupNATSContainer(t *testing.T) (testcontainers.Container, string) {
	ctx := context.Background()

	natsContainer, err := natscontainer.RunContainer(ctx,
		testcontainers.WithImage("nats:2.10-alpine"),
	)
	require.NoError(t, err, "Failed to start NATS container")

	connectionString, err := natsContainer.ConnectionString(ctx)
	require.NoError(t, err, "Failed to get NATS connection string")

	return natsContainer, connectionString
}

func TestEventLogger_HandleEvent(t *testing.T) {
	tests := []struct {
		name           string
		messageData    []byte
		expectedFields map[string]interface{}
		expectedLevel  logrus.Level
	}{
		{
			name:        "valid JSON event",
			messageData: createValidEventData(),
			expectedFields: map[string]interface{}{
				"event_id":   "test-123",
				"event_type": "user.created",
				"source":     "user-service",
				"subject":    "test.subject",
			},
			expectedLevel: logrus.InfoLevel,
		},
		{
			name:        "invalid JSON - raw message",
			messageData: []byte("this is not json"),
			expectedFields: map[string]interface{}{
				"subject":  "test.subject",
				"raw_data": "this is not json",
			},
			expectedLevel: logrus.InfoLevel,
		},
		{
			name:        "empty message",
			messageData: []byte(""),
			expectedFields: map[string]interface{}{
				"subject":  "test.subject",
				"raw_data": "",
			},
			expectedLevel: logrus.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runHandleEventTest(t, tt.messageData, tt.expectedFields, tt.expectedLevel)
		})
	}
}

func createValidEventData() []byte {
	event := nats.Event{
		ID:        "test-123",
		Type:      "user.created",
		Source:    "user-service",
		Timestamp: time.Date(2025, 1, 11, 10, 30, 0, 0, time.UTC),
		Data: map[string]interface{}{
			"user_id": "12345",
			"email":   "test@example.com",
		},
	}
	data, _ := json.Marshal(event)
	return data
}

func runHandleEventTest(
	t *testing.T,
	messageData []byte,
	expectedFields map[string]interface{},
	expectedLevel logrus.Level,
) {
	logger, hook := test.NewNullLogger()
	eventLogger := nats.NewEventLogger(logger)

	msg := &natsclient.Msg{
		Subject: "test.subject",
		Data:    messageData,
		Reply:   "test.reply",
	}

	err := eventLogger.HandleEvent(msg)
	require.NoError(t, err)

	assert.Len(t, hook.Entries, 1)
	entry := hook.Entries[0]
	assert.Equal(t, expectedLevel, entry.Level)

	for key, expectedValue := range expectedFields {
		actualValue, exists := entry.Data[key]
		assert.True(t, exists, "Expected field %s not found in log entry", key)
		assert.Equal(t, expectedValue, actualValue)
	}
}

func TestEventLogger_HandleRawEvent(t *testing.T) {
	testCases := []struct {
		name        string
		messageData []byte
		subject     string
		reply       string
	}{
		{
			name:        "simple text message",
			messageData: []byte("Hello, JetStream World!"),
			subject:     "test.subject",
			reply:       "test.reply",
		},
		{
			name:        "binary data",
			messageData: []byte{0x00, 0x01, 0x02, 0xFF},
			subject:     "binary.subject",
			reply:       "",
		},
		{
			name:        "empty message",
			messageData: []byte(""),
			subject:     "empty.subject",
			reply:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runHandleRawEventTest(t, tc.messageData, tc.subject, tc.reply)
		})
	}
}

func runHandleRawEventTest(t *testing.T, messageData []byte, subject, reply string) {
	logger, hook := test.NewNullLogger()
	eventLogger := nats.NewEventLogger(logger)

	msg := &natsclient.Msg{
		Subject: subject,
		Data:    messageData,
		Reply:   reply,
	}

	err := eventLogger.HandleRawEvent(msg)
	require.NoError(t, err)

	assert.Len(t, hook.Entries, 1)
	entry := hook.Entries[0]
	assert.Equal(t, logrus.InfoLevel, entry.Level)

	verifyRequiredFields(t, entry)
	verifySpecificValues(t, entry, messageData, subject, reply)
}

func verifyRequiredFields(t *testing.T, entry logrus.Entry) {
	requiredFields := []string{"subject", "data", "data_size", "reply", "timestamp"}

	for _, field := range requiredFields {
		_, exists := entry.Data[field]
		assert.True(t, exists, "Required field %s not found in log entry", field)
	}
}

func verifySpecificValues(t *testing.T, entry logrus.Entry, messageData []byte, subject, reply string) {
	assert.Equal(t, subject, entry.Data["subject"])
	assert.Equal(t, string(messageData), entry.Data["data"])
	assert.Equal(t, len(messageData), entry.Data["data_size"])
	assert.Equal(t, reply, entry.Data["reply"])
}

func TestEventLogger_HandleEventWithCustomFields(t *testing.T) {
	tests := []struct {
		name           string
		messageData    []byte
		expectedFields []string
	}{
		{
			name:           "valid JSON with custom fields",
			messageData:    createCustomFieldsData(),
			expectedFields: []string{"subject", "data_size", "reply", "timestamp", "custom_field_1", "custom_field_2", "nested"},
		},
		{
			name:           "non-JSON data",
			messageData:    []byte("not json"),
			expectedFields: []string{"subject", "data_size", "reply", "timestamp", "raw_data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCustomFieldsTest(t, tt.messageData, tt.expectedFields)
		})
	}
}

func createCustomFieldsData() []byte {
	data := map[string]interface{}{
		"custom_field_1": "value1",
		"custom_field_2": 42,
		"nested": map[string]interface{}{
			"key": "value",
		},
	}
	jsonData, _ := json.Marshal(data)
	return jsonData
}

func runCustomFieldsTest(t *testing.T, messageData []byte, expectedFields []string) {
	logger, hook := test.NewNullLogger()
	eventLogger := nats.NewEventLogger(logger)

	msg := &natsclient.Msg{
		Subject: "test.subject",
		Data:    messageData,
		Reply:   "test.reply",
	}

	err := eventLogger.HandleEventWithCustomFields(msg)
	require.NoError(t, err)

	assert.Len(t, hook.Entries, 1)
	entry := hook.Entries[0]

	for _, field := range expectedFields {
		_, exists := entry.Data[field]
		assert.True(t, exists, "Expected field %s not found in log entry", field)
	}
}

func TestNewEventLogger(t *testing.T) {
	t.Run("with provided logger", func(t *testing.T) {
		logger := logrus.New()
		eventLogger := nats.NewEventLogger(logger)

		assert.NotNil(t, eventLogger)
	})

	t.Run("with nil logger", func(t *testing.T) {
		eventLogger := nats.NewEventLogger(nil)

		assert.NotNil(t, eventLogger)
	})
}

func TestEventLogger_IntegrationWithNATS(t *testing.T) {
	natsContainer, connectionString := setupNATSContainer(t)
	defer func() {
		assert.NoError(t, natsContainer.Terminate(context.Background()))
	}()

	nc, err := natsclient.Connect(connectionString)
	require.NoError(t, err)
	defer nc.Close()

	logger, hook := test.NewNullLogger()
	eventLogger := nats.NewEventLogger(logger)

	testEvent := createIntegrationTestEvent()
	eventData, err := json.Marshal(testEvent)
	require.NoError(t, err)

	subject := "test.integration"
	sub, err := nc.Subscribe(subject, func(msg *natsclient.Msg) {
		handlerErr := eventLogger.HandleEvent(msg)
		assert.NoError(t, handlerErr)
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, sub.Unsubscribe())
	}()

	err = nc.Publish(subject, eventData)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.Len(t, hook.Entries, 1)
	entry := hook.Entries[0]
	assert.Equal(t, logrus.InfoLevel, entry.Level)
	assert.Equal(t, subject, entry.Data["subject"])
	assert.Equal(t, testEvent.ID, entry.Data["event_id"])
	assert.Equal(t, testEvent.Type, entry.Data["event_type"])
	assert.Equal(t, testEvent.Source, entry.Data["source"])
}

func createIntegrationTestEvent() nats.Event {
	return nats.Event{
		ID:        "integration-test-123",
		Type:      "integration.test",
		Source:    "test-suite",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"test":    true,
			"message": "integration test",
		},
	}
}

func TestEventLogger_IntegrationWithRawMessage(t *testing.T) {
	natsContainer, connectionString := setupNATSContainer(t)
	defer func() {
		assert.NoError(t, natsContainer.Terminate(context.Background()))
	}()

	nc, err := natsclient.Connect(connectionString)
	require.NoError(t, err)
	defer nc.Close()

	logger, hook := test.NewNullLogger()
	eventLogger := nats.NewEventLogger(logger)

	rawMessage := "This is a raw integration test message"
	subject := "test.raw.integration"

	sub, err := nc.Subscribe(subject, func(msg *natsclient.Msg) {
		handlerErr := eventLogger.HandleRawEvent(msg)
		assert.NoError(t, handlerErr)
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, sub.Unsubscribe())
	}()

	err = nc.Publish(subject, []byte(rawMessage))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.Len(t, hook.Entries, 1)
	entry := hook.Entries[0]
	assert.Equal(t, logrus.InfoLevel, entry.Level)
	assert.Equal(t, subject, entry.Data["subject"])
	assert.Equal(t, rawMessage, entry.Data["data"])
	assert.Equal(t, len(rawMessage), entry.Data["data_size"])
}

func BenchmarkEventLogger_HandleEvent(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	eventLogger := nats.NewEventLogger(logger)

	event := nats.Event{
		ID:        "bench-test",
		Type:      "benchmark.jetstream.test",
		Source:    "benchmark",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"iteration": 0,
			"benchmark": true,
		},
	}
	eventData, _ := json.Marshal(event)

	msg := &natsclient.Msg{
		Subject: "benchmark.subject",
		Data:    eventData,
		Reply:   "",
	}

	b.ResetTimer()
	for range b.N {
		_ = eventLogger.HandleEvent(msg)
	}
}

func BenchmarkEventLogger_HandleRawEvent(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	eventLogger := nats.NewEventLogger(logger)

	msg := &natsclient.Msg{
		Subject: "benchmark.subject",
		Data:    []byte("benchmark raw jetstream message"),
		Reply:   "",
	}

	b.ResetTimer()
	for range b.N {
		_ = eventLogger.HandleRawEvent(msg)
	}
}
