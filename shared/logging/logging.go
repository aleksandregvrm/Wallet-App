package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

type ctxKey struct{}

var requestIdStructure = ctxKey{}

// injecting the logging identifier in the context
// Reusable by the caller after the injection
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIdStructure, requestID)
}

type InternalLogger struct {
	slog *slog.Logger
}

func NewInternalLogger() *InternalLogger {
	return &InternalLogger{
		slog: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
}

// Each of the identifier will be server withing the context struct and retrieve from there for relevant logging
// The unique identifier for the relevant log will be inserted by the caller. within the context with the property name of request_id
func (l *InternalLogger) requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Asserting the string type since before that we would get the nil pointer error
	// Also to retrieve the value from the context it is essential to have a struct
	requestID, _ := ctx.Value(requestIdStructure).(string)

	// Fallback in case there's no injected identifier - meaning we have more generalized log
	if requestID == "" {
		return "General"
	}

	return requestID
}

func (l *InternalLogger) log(ctx context.Context, level slog.Level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	l.slog.Log(ctx, level, msg, "log-identifier", l.requestIDFromContext(ctx))
}

func (l *InternalLogger) LogWarn(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, slog.LevelWarn, format, args...)
}

func (l *InternalLogger) LogInfo(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, slog.LevelInfo, format, args...)
}

func (l *InternalLogger) LogError(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, slog.LevelError, format, args...)
}

func (l *InternalLogger) LogDebug(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, slog.LevelDebug, format, args...)
}

func (l *InternalLogger) LogFatal(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, slog.LevelError, format, args...)
	// Ending the process within os scope
	os.Exit(1)
}
