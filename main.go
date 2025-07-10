package main

import (
	"context"
	"net/http"
	"os"

	"events-audit/internal/constants"
	"events-audit/internal/server"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/juju/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var (
	version = "dev"
)

func beforeAction(ctx context.Context, c *cli.Command) (context.Context, error) {
	levelParam := c.String("log-level")
	formatParam := c.String("log-format")
	if formatParam != "text" && formatParam != "json" {
		return ctx, errors.New("log format suppoorte only json or text")
	}

	if formatParam == "text" {
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "02.01.2006 15:04:05.000",
			FullTimestamp:   true,
		})
	}
	if formatParam == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "02.01.2006 15:04:05.000",
		})
	}

	level, err := logrus.ParseLevel(levelParam)
	if err != nil {
		return ctx, err
	}

	logrus.SetLevel(level)

	return ctx, nil
}

func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, map[string]any{
			"Status": "OK",
		})
	}
}
func startHealth(addr string) error {

	r := chi.NewRouter()

	r.Get("/health", healthHandler())

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	go func() {

		err := srv.ListenAndServe()
		if err != nil {
			logrus.WithError(err).Error("failed server start")
		}
	}()
	return nil

}
func mainAction(ctx context.Context, c *cli.Command) error {

	listenAddr := c.String("health-addr")
	// Check if audit is enabled
	auditType := c.String("audit")
	if auditType == "nope" {
		logrus.Info("Audit is disabled, exiting")
		return nil
	}

	if auditType != "nats" {
		return errors.New("only NATS audit is currently supported")
	}

	natsAddr := c.String("audit-nats-addr")
	if natsAddr == "" {
		return errors.New("NATS address is required when audit type is 'nats'")
	}

	// Create server configuration
	config := server.Config{
		NatsURL:         natsAddr,
		NatsSubject:     c.String("audit-topic"),
		StreamName:      c.String("audit-stream-name"),
		ConsumerName:    c.String("audit-consumer-name"),
		DurableName:     c.String("audit-durable-name"),
		CreateStream:    c.Bool("audit-create-stream"),
		MaxDeliver:      c.Int("audit-max-deliver"),
		AckWait:         c.Duration("audit-ack-wait"),
		PullMaxMessages: c.Int("audit-pull-max-messages"),
		PullTimeout:     c.Duration("audit-pull-timeout"),
		StreamMaxAge:    c.Duration("audit-stream-max-age"),
		StreamMaxBytes:  c.Int64("audit-stream-max-bytes"),
		StreamMaxMsgs:   c.Int64("audit-stream-max-msgs"),
		StreamReplicas:  c.Int("audit-stream-replicas"),
		LogLevel:        c.String("log-level"),
		LogFormat:       c.String("log-format"),
	}
	err := startHealth(listenAddr)
	if err != nil {
		return nil
	}

	// Create and run the server
	srv := server.NewServer(config)
	return srv.Run(ctx)
}

func createBaseFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "log-level",
			Usage:    "logger verbosity `LEVEL`",
			Value:    "debug",
			Sources:  cli.EnvVars("AUDIT_LISTNER_LOG_LEVEL"),
			Category: "base",
		},
		&cli.StringFlag{
			Name:     "log-format",
			Usage:    "logger format `text` or `json`",
			Value:    "text",
			Sources:  cli.EnvVars("AUDIT_LISTNER_LOG_FORMAT"),
			Category: "base",
		},

		&cli.StringFlag{
			Name:     "health-addr",
			Usage:    "health addr ",
			Value:    ":3000",
			Sources:  cli.EnvVars("AUDIT_LISTNER_HEALTH_ADDR"),
			Category: "base",
		},
	}
}

func createAuditFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "audit",
			Usage:    "enable audit, supported nats, nsq, nope",
			Value:    "nope",
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT"),
			Category: "audit",
		},
		&cli.StringFlag{
			Name:     "audit-topic",
			Usage:    "topic `TOPIC`",
			Value:    "accountats",
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_TOPIC"),
			Category: "audit",
		},
		&cli.StringFlag{
			Name:     "audit-nats-addr",
			Usage:    "nats addr `ADDRESS`",
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_NATS_ADDR"),
			Category: "audit",
		},
	}
}

func createJetStreamFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "audit-stream-name",
			Usage:    "JetStream stream name `NAME`",
			Value:    "EVENTS",
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_STREAM_NAME"),
			Category: "jetstream",
		},
		&cli.StringFlag{
			Name:     "audit-consumer-name",
			Usage:    "JetStream consumer name `NAME`",
			Value:    "events-audit-consumer",
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_CONSUMER_NAME"),
			Category: "jetstream",
		},
		&cli.StringFlag{
			Name:     "audit-durable-name",
			Usage:    "JetStream durable consumer name `NAME`",
			Value:    "events-audit-durable",
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_DURABLE_NAME"),
			Category: "jetstream",
		},
		&cli.BoolFlag{
			Name:     "audit-create-stream",
			Usage:    "create JetStream stream if it doesn't exist",
			Value:    true,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_CREATE_STREAM"),
			Category: "jetstream",
		},
		&cli.IntFlag{
			Name:     "audit-max-deliver",
			Usage:    "maximum delivery attempts for messages `COUNT`",
			Value:    constants.DefaultMaxDeliver,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_MAX_DELIVER"),
			Category: "jetstream",
		},
		&cli.DurationFlag{
			Name:     "audit-ack-wait",
			Usage:    "acknowledgment wait time `DURATION`",
			Value:    constants.DefaultAckWait,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_ACK_WAIT"),
			Category: "jetstream",
		},
		&cli.IntFlag{
			Name:     "audit-pull-max-messages",
			Usage:    "maximum messages to pull at once `COUNT`",
			Value:    constants.DefaultPullMaxMessages,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_PULL_MAX_MESSAGES"),
			Category: "jetstream",
		},
		&cli.DurationFlag{
			Name:     "audit-pull-timeout",
			Usage:    "pull timeout `DURATION`",
			Value:    constants.DefaultPullTimeout,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_PULL_TIMEOUT"),
			Category: "jetstream",
		},
		&cli.DurationFlag{
			Name:     "audit-stream-max-age",
			Usage:    "maximum age for messages in stream `DURATION`",
			Value:    constants.DefaultStreamMaxAge,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_STREAM_MAX_AGE"),
			Category: "jetstream",
		},
		&cli.Int64Flag{
			Name:     "audit-stream-max-bytes",
			Usage:    "maximum bytes for stream `BYTES`",
			Value:    constants.DefaultStreamMaxBytes,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_STREAM_MAX_BYTES"),
			Category: "jetstream",
		},
		&cli.Int64Flag{
			Name:     "audit-stream-max-msgs",
			Usage:    "maximum messages for stream `COUNT`",
			Value:    constants.DefaultStreamMaxMsgs,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_STREAM_MAX_MSGS"),
			Category: "jetstream",
		},
		&cli.IntFlag{
			Name:     "audit-stream-replicas",
			Usage:    "number of stream replicas `COUNT`",
			Value:    1,
			Sources:  cli.EnvVars("AUDIT_LISTNER_AUDIT_STREAM_REPLICAS"),
			Category: "jetstream",
		},
	}
}

func createAllFlags() []cli.Flag {
	var flags []cli.Flag
	flags = append(flags, createBaseFlags()...)
	flags = append(flags, createAuditFlags()...)
	flags = append(flags, createJetStreamFlags()...)
	return flags
}

func runApplication() error {
	cmd := &cli.Command{
		Name:    "audit-listner",
		Usage:   "audit-listner",
		Before:  beforeAction,
		Action:  mainAction,
		Version: version,
		Flags:   createAllFlags(),
	}

	return cmd.Run(context.Background(), os.Args)
}

func main() {
	if err := runApplication(); err != nil {
		logrus.WithError(err).Error("Application failed to start")
		os.Exit(1)
	}
}
