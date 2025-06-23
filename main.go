package main

import (
	"context"
	"os"

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

func mainAction(ctx context.Context, c *cli.Command) error {
	// listenAddr := c.String("endpoint")
	// databaseDSN := c.String("database-dsn")
	// privateKey := c.String("private-key")
	// publicKey := c.String("public-key")

	// return server.Run()
	return nil
}

func main() {
	cmd := &cli.Command{
		Name:    "audit-listner",
		Usage:   "audit-listner",
		Before:  beforeAction,
		Action:  mainAction,
		Version: version,
		Flags: []cli.Flag{
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
		},
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		logrus.WithError(err).Error("Application failed to start")
		os.Exit(1)
	}
}
