package appkit

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"

	charmlog "github.com/charmbracelet/log"
)

// Logger interface for appkit...
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

type LoggerType string

const (
	LoggerCharm LoggerType = "charm" // colored text logging
	LoggerSlog  LoggerType = "slog"  // structured JSON output
)

type LoggerConfig struct {
	Level      string     
	Prefix     string     
	LoggerType LoggerType 
}

// InitLogger returns a Logger based on the config.
// In dev, use LoggerCharm for colored output.
// In prod/cloud, use LoggerSlog for structured JSON.
func InitLogger(cfg LoggerConfig) Logger {
	cfg.Level = strings.ToLower(cfg.Level)

	switch cfg.LoggerType {
	case LoggerSlog:
		return newSlogLogger(cfg)
	default:
		return newCharmLogger(cfg)
	}
}

////////////////////////////////////
// Charm Logger implementation

type charmLogger struct {
	l *charmlog.Logger
}

func newCharmLogger(cfg LoggerConfig) Logger {
	l := charmlog.New(os.Stdout)
	l.SetPrefix(cfg.Prefix)
	l.SetLevel(charmLevel(cfg.Level))
	return &charmLogger{l: l}
}

func charmLevel(level string) charmlog.Level {
	switch level {
	case "debug":
		return charmlog.DebugLevel
	case "info":
		return charmlog.InfoLevel
	case "warn":
		return charmlog.WarnLevel
	default:
		return charmlog.ErrorLevel
	}
}

func (c *charmLogger) Debug(msg string, args ...any)            { c.l.Debug(msg, args...) }
func (c *charmLogger) Info(msg string, args ...any)             { c.l.Info(msg, args...) }
func (c *charmLogger) Warn(msg string, args ...any)             { c.l.Warn(msg, args...) }
func (c *charmLogger) Error(msg string, args ...any)            { c.l.Error(msg, args...) }
func (c *charmLogger) Debugf(format string, args ...any)        { c.l.Debugf(format, args...) }
func (c *charmLogger) Infof(format string, args ...any)         { c.l.Infof(format, args...) }
func (c *charmLogger) Warnf(format string, args ...any)         { c.l.Warnf(format, args...) }
func (c *charmLogger) Errorf(format string, args ...any)        { c.l.Errorf(format, args...) }

////////////////////////////////////
// slog implementation
// Uses JSON handler for structured prod logging.
// Swap the handler here if you want tint (colored slog)

type slogLogger struct {
	l      *slog.Logger
	level  slog.Level
	prefix string
}

func newSlogLogger(cfg LoggerConfig) Logger {
	level := slogLevel(cfg.Level)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	l := slog.New(handler)
	if cfg.Prefix != "" {
		l = l.With("app", cfg.Prefix)
	}
	return &slogLogger{l: l, level: level, prefix: cfg.Prefix}
}

func slogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

func (s *slogLogger) Debug(msg string, args ...any)     { s.l.Debug(msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)      { s.l.Info(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)      { s.l.Warn(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any)     { s.l.Error(msg, args...) }
func (s *slogLogger) Debugf(format string, args ...any) { s.l.Debug(fmt.Sprintf(format, args...)) }
func (s *slogLogger) Infof(format string, args ...any)  { s.l.Info(fmt.Sprintf(format, args...)) }
func (s *slogLogger) Warnf(format string, args ...any)  { s.l.Warn(fmt.Sprintf(format, args...)) }
func (s *slogLogger) Errorf(format string, args ...any) { s.l.Error(fmt.Sprintf(format, args...)) }


// Logging helpers...

// LogConfig generically logs all fields of a config struct,
// masking fields tagged with `log:"secret"`
func LogConfig(cfg any, logger Logger) {
	v := reflect.ValueOf(cfg)
	t := reflect.TypeOf(cfg)

	if v.Kind() == reflect.Pointer {
		v = v.Elem()
		t = t.Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		display := fmt.Sprintf("%v", value.Interface())
		if field.Tag.Get("log") == "secret" {
			display = "********"
		}
		logger.Infof("%-20s %v", field.Name+":", display)
	}
}
