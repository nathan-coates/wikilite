package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// LogLevel defines the severity of a log entry.
type LogLevel string

const (
	// LevelInfo is for general operational entries.
	LevelInfo LogLevel = "INFO"
	// LevelWarning is for non-critical issues.
	LevelWarning LogLevel = "WARNING"
	// LevelError is for errors that should be investigated.
	LevelError LogLevel = "ERROR"
	// LevelDebug is for detailed debug information.
	LevelDebug LogLevel = "DEBUG"
	// LevelSQL is for SQL queries.
	LevelSQL LogLevel = "SQL"
	// LevelSQLError is for SQL queries that result in an error.
	LevelSQLError LogLevel = "SQL_ERROR"
)

// SystemLog represents a single log entry.
type SystemLog struct {
	bun.BaseModel `bun:"table:system_logs"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	Level     LogLevel  `bun:"level"                                                 json:"level"`
	Source    string    `bun:"source"                                                json:"source"`
	Message   string    `bun:"message"                                               json:"message"`
	Data      string    `bun:"data,type:text"                                        json:"data"`

	Id       int64 `bun:"id,pk,autoincrement" json:"id"`
	Duration int64 `bun:"duration_ms"         json:"durationMs"`
}

// Logger is a function that logs a message.
type Logger func(ctx context.Context, level LogLevel, source string, message string, data string) error

// contextKey is a private type to prevent key collisions
type contextKey string

const loggerContextKey contextKey = "db_logger"

// NewContextWithLogger creates a new context containing the logger.
func NewContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// LoggerFromContext retrieves the logger from the context, if it exists.
func LoggerFromContext(ctx context.Context) Logger {
	logger, ok := ctx.Value(loggerContextKey).(Logger)
	if !ok {
		return nil
	}
	return logger
}
