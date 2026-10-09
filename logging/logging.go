// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package logging

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/immanent-tech/go-base/config"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	slogmulti "github.com/samber/slog-multi"
	slogctx "github.com/veqryn/slog-context"
	slogotel "github.com/veqryn/slog-context/otel"
)

const (
	configEnvPrefix = "LOG_"

	// LevelTrace is a custom TRACE log level.
	LevelTrace = slog.Level(-8)
	// LevelNotice is a custom NOTICE log level.
	LevelNotice = slog.Level(2) // between INFO (0) and WARN (4)
	// LevelFatal is a custom FATAL log level.
	LevelFatal = slog.Level(12)
	// LevelCritical is an alias of [LevelFatal].
	LevelCritical = LevelFatal
)

// LevelNames contains a list of custom log level names.
var LevelNames = map[slog.Leveler]string{
	LevelTrace:  "TRACE",
	LevelNotice: "NOTICE",
	LevelFatal:  "FATAL",
}

type Config struct {
	// Level is the current default logging level.
	Level string `koanf:"level" validate:"omitempty,oneof=trace debug info warn error"`
	// Format overrides any auto-assigned format of the logs to the one specified.
	Format string `koanf:"format" validate:"omitempty,oneof=json console"`
	// LogFile is a file to which logs will be written. Any existing file will be overwritten.
	LogFile      string     `koanf:"file" validate:"omitempty,file"`
	currentLevel slog.Level `koanf:"-"`
}

var cfg *Config

// New creates a new logger with the given options.
var New = sync.OnceValue(func() *slog.Logger {
	// Read config from environment.
	cfg = &Config{
		Level:        "info",
		currentLevel: slog.LevelInfo,
	}
	if err := config.Load(configEnvPrefix, cfg); err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	// Set the log level.
	switch cfg.Level {
	case "trace":
		cfg.currentLevel = LevelTrace
	case "debug":
		cfg.currentLevel = slog.LevelDebug
	case "info":
		cfg.currentLevel = slog.LevelInfo
	case "warn":
		cfg.currentLevel = slog.LevelWarn
	case "error":
		cfg.currentLevel = slog.LevelError
	default:
		cfg.currentLevel = slog.LevelInfo
	}

	var handlers []slog.Handler

	// When logging in a container, use JSON output and disable logfile, unless the "console" format has been specified.
	if config.DetectContainerRuntime() != config.RuntimeNone && cfg.Format != "console" {
		slog.Info("Using container logging format.")
		cfg.LogFile = ""
		instrumentedHandler := HandlerWithSpanContext(
			slog.NewJSONHandler(os.Stderr, containerConsoleOptions(cfg.currentLevel)),
		)
		handlers = append(handlers,
			instrumentedHandler,
		)
	} else {
		slog.Info("Using text logging format.")
		handlers = append(handlers,
			tint.NewTextHandler(os.Stderr, consoleOptions(cfg.currentLevel, os.Stderr.Fd())),
		)
	}

	// Unless no logfile was requested, set up file logging.
	if cfg.LogFile != "" {
		if logFH, err := openLogFile(cfg.LogFile); err != nil {
			fmt.Fprintln(os.Stderr, "unable to open log file: %w", err)
		} else {
			handlers = append(handlers,
				slog.NewTextHandler(logFH, generateFileOpts(cfg.currentLevel)),
			)
		}
	}

	logger := slog.New(slogctx.NewHandler(slogmulti.Fanout(handlers...), &slogctx.HandlerOptions{
		// Prependers will first add the OTEL Trace ID,
		// then anything else Prepended to the ctx
		Prependers: []slogctx.AttrExtractor{
			slogotel.ExtractTraceSpanID,
			slogctx.ExtractPrepended,
		},
		// Appenders stays as default (leaving as nil would accomplish the same)
		Appenders: []slogctx.AttrExtractor{
			slogctx.ExtractAppended,
		},
	}))
	slog.SetDefault(logger)

	logger.Info("Logger initialised.")

	return logger
})

// GetLogLevel returns the current default log level.
func GetLogLevel() slog.Level {
	return cfg.currentLevel
}

func containerConsoleOptions(level slog.Level) *slog.HandlerOptions {
	opts := &slog.HandlerOptions{
		AddSource:   false,
		Level:       level,
		ReplaceAttr: containerReplacer,
	}
	if level == LevelTrace {
		opts.AddSource = true
	}
	return opts
}

func consoleOptions(level slog.Level, fd uintptr) *tint.Options {
	opts := &tint.Options{
		Level:       level,
		NoColor:     !isatty.IsTerminal(fd),
		ReplaceAttr: consolelevelReplacer,
		TimeFormat:  time.Kitchen,
	}
	if level == LevelTrace {
		opts.AddSource = true
	}

	return opts
}

func generateFileOpts(level slog.Level) *slog.HandlerOptions {
	opts := &slog.HandlerOptions{
		AddSource:   false,
		Level:       level,
		ReplaceAttr: fileLevelReplacer,
	}
	if level == LevelTrace {
		opts.AddSource = true
	}

	return opts
}

func consolelevelReplacer(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.LevelKey {
		level, ok := attr.Value.Any().(slog.Level)
		if !ok {
			level = slog.LevelInfo
		}
		switch level {
		case slog.LevelError:
			attr.Value = slog.StringValue(color.HiRedString("ERROR"))
		case slog.LevelWarn:
			attr.Value = slog.StringValue(color.HiYellowString("WARN"))
		case slog.LevelInfo:
			attr.Value = slog.StringValue(color.HiGreenString("INFO"))
		case slog.LevelDebug:
			attr.Value = slog.StringValue(color.HiMagentaString("DEBUG"))
		case LevelTrace:
			attr.Value = slog.StringValue(color.HiWhiteString("TRACE"))
		default:
			attr.Value = slog.StringValue("UNKNOWN")
		}
	}

	return attr
}

func fileLevelReplacer(_ []string, attr slog.Attr) slog.Attr {
	// Set default level.
	if attr.Key == slog.LevelKey {
		level, ok := attr.Value.Any().(slog.Level)
		if !ok {
			level = slog.LevelInfo
		}

		// Format custom log level.
		if levelLabel, exists := LevelNames[level]; exists {
			attr.Value = slog.StringValue(levelLabel)
		}
	}

	return attr
}

// ReplaceAttr replaces slog default attributes with GCP compatible ones
// https://cloud.google.com/logging/docs/structured-logging
// https://cloud.google.com/logging/docs/agent/logging/configuration#special-fields
func containerReplacer(groups []string, attr slog.Attr) slog.Attr {
	// Only rename top-level keys.
	if len(groups) > 0 {
		return attr
	}
	switch attr.Key {
	case slog.LevelKey:
		attr.Key = "severity"
		if lvl, ok := attr.Value.Any().(slog.Level); ok {
			switch {
			case lvl >= LevelCritical:
				attr.Value = slog.StringValue("CRITICAL")
			case lvl >= slog.LevelError:
				attr.Value = slog.StringValue("ERROR")
			case lvl >= slog.LevelWarn:
				attr.Value = slog.StringValue("WARNING")
			case lvl >= LevelNotice:
				attr.Value = slog.StringValue("NOTICE")
			case lvl >= slog.LevelInfo:
				attr.Value = slog.StringValue("INFO")
			default:
				attr.Value = slog.StringValue("DEBUG")
			}
		}
	case slog.MessageKey:
		attr.Key = "message"
	case slog.SourceKey:
		attr.Key = "logging.googleapis.com/sourceLocation"
		if src, ok := attr.Value.Any().(*slog.Source); ok {
			attr.Value = slog.GroupValue(
				slog.String("file", src.File),
				slog.String("line", strconv.Itoa(src.Line)),
				slog.String("function", src.Function),
			)
		}
	}
	return attr
}

// openLogFile will attempt to open the specified log file. It will also attempt
// to create the directory containing the log file if it does not exist.
func openLogFile(logFile string) (*os.File, error) {
	logDir := filepath.Dir(logFile)
	// Create the log directory if it does not exist.
	if _, err := os.Stat(logDir); err == nil || errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(logDir, 0o750)
		if err != nil {
			return nil, fmt.Errorf("unable to create log file directory %s: %w", logDir, err)
		}
	}

	// Open the log file.
	logFileHandle, err := os.Create(logFile) // #nosec:G304
	if err != nil {
		return nil, fmt.Errorf("unable to open log file: %w", err)
	}

	return logFileHandle, nil
}
