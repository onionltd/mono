package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

func run() error {
	cfg, err := setupConfig()
	if err != nil {
		return err
	}

	rootLogger, err := setupLogger(cfg)
	if err != nil {
		return err
	}

	armoredKeyRing, err := readArmoredKeyRing(cfg)
	if err != nil {
		return err
	}

	repo, err := gitCloneOrOpen(cfg)
	if err != nil {
		return err
	}

	if !cfg.NoVerifySignature {
		if err := gitVerifyHeadCommitSignature(repo, armoredKeyRing); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		t := time.Duration(0)
		for {
			select {
			case <-time.After(t):
				if err := gitPullChanges(repo); err != nil {
					rootLogger.Info("pull changes", zap.Error(err))
				}
				if !cfg.NoVerifySignature {
					if err := gitVerifyHeadCommitSignature(repo, armoredKeyRing); err != nil {
						rootLogger.Error("verify commit signature", zap.Error(err))
						return
					}
				}
				t = cfg.SyncInterval

			case <-ctx.Done():
				return
			}
		}
	}()

	// Handle termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		rootLogger.Warn("received a termination signal")
		cancel()
	}()

	wg.Wait()
	return nil
}

func setupConfig() (*config, error) {
	cfg := &config{}
	parser := flags.NewParser(cfg, flags.HelpFlag)
	if _, err := parser.Parse(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func setupLogger(cfg *config) (*zap.Logger, error) {
	translateLogLevel := func(s string) zapcore.Level {
		switch strings.ToLower(s) {
		case "debug":
			return zap.DebugLevel
		case "info":
			return zap.InfoLevel
		case "warning":
			return zap.WarnLevel
		case "error":
			return zap.ErrorLevel
		default:
			return zap.InfoLevel
		}
	}
	logLevel := translateLogLevel(cfg.LogLevel)
	return zap.Config{
		Encoding:         "console",
		Level:            zap.NewAtomicLevelAt(logLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			NameKey:    "name",
			EncodeName: zapcore.FullNameEncoder,
		},
	}.Build()
}

func readArmoredKeyRing(cfg *config) (string, error) {
	if cfg.NoVerifySignature {
		return "", nil
	}
	if cfg.KeyRingPath == "" {
		return "", errors.New("keyring not specified")
	}
	b, err := ioutil.ReadFile(cfg.KeyRingPath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
