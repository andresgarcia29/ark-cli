// Package logs provides ark's diagnostic log. It is separate from user-facing
// output: everything here is for debugging and stays silent unless asked for.
package logs

import (
	"sync"

	"github.com/andresgarcia29/ark-cli/lib/ui"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.SugaredLogger
	once   sync.Once
	level  = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
)

// Init configures the diagnostic log. Without debug it stays effectively
// silent, because user-facing messages are the ui package's job.
func Init(debug bool) {
	once.Do(build)
	if debug {
		level.SetLevel(zapcore.DebugLevel)
	} else {
		// Errors already travel back to the caller as values and get reported
		// once by the command layer; logging them again would duplicate them.
		level.SetLevel(zapcore.FatalLevel)
	}
}

func build() {
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")

	// Diagnostics go to stderr so stdout stays parseable, and stack traces are
	// left out because they are noise for a CLI user.
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(ui.Err),
		level,
	)
	logger = zap.New(core).Sugar()
}

// GetLogger returns the diagnostic logger.
func GetLogger() *zap.SugaredLogger {
	once.Do(build)
	return logger
}

// Sync flushes buffered entries before exit.
func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}
