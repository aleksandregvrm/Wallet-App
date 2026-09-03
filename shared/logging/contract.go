package logging

import "context"

// Defines a structure for our internal Logger to make sure services do not depend on concrete logging libraries
// But rather on our abstraction of those libraries. This make logging interchangeable without coupling library directly to the service
type Logger interface {
	LogWarn(ctx context.Context, format string, args ...interface{})
	LogInfo(ctx context.Context, format string, args ...interface{})
	LogError(ctx context.Context, format string, args ...interface{})
	LogDebug(ctx context.Context, format string, args ...interface{})
	LogFatal(ctx context.Context, format string, args ...interface{})
}
