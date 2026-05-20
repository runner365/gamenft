package logger

import (
	"sync"
	"testing"
)

// resetGlobals resets package-level globals between tests.
func resetGlobals() {
	once = sync.Once{}
	logger = nil
}

func TestInitLoggerAndLogMethods(t *testing.T) {
	resetGlobals()

	cfg := Config{Level: "info", Filename: "test.log", MaxSize: 1000, MaxBackups: 3, MaxAge: 7, Compress: true}
	InitLogger(cfg)

	if logger == nil {
		t.Fatal("logger should be initialized after InitLogger")
	}

	// Ensure logging functions don't panic and will initialize if needed
	LogInfof("test info: %s", "ok")
	LogDebugf("test debug: %s", "ok")
	LogWarnf("test warn: %s", "ok")
	LogErrorf("test error: %s", "ok")

	Close()
	resetGlobals()
}

func TestCloseWithoutInit(t *testing.T) {
	resetGlobals()
	// Should not panic when logger is nil
	Close()
}
