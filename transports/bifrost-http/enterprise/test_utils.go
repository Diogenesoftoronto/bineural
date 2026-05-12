package enterprise

import (
	"sync"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

// MockLogger implements schemas.Logger for testing
type MockLogger struct {
	mu       sync.Mutex
	logs     []string
	errors   []string
	debugs   []string
	infos    []string
	warnings []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs:     make([]string, 0),
		errors:   make([]string, 0),
		debugs:   make([]string, 0),
		infos:    make([]string, 0),
		warnings: make([]string, 0),
	}
}

func (ml *MockLogger) SetLevel(level schemas.LogLevel) {}

func (ml *MockLogger) SetOutputType(outputType schemas.LoggerOutputType) {}

func (ml *MockLogger) Error(format string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.errors = append(ml.errors, format)
}

func (ml *MockLogger) Warn(format string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.warnings = append(ml.warnings, format)
}

func (ml *MockLogger) Info(format string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.infos = append(ml.infos, format)
}

func (ml *MockLogger) Debug(format string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.debugs = append(ml.debugs, format)
}

func (ml *MockLogger) Fatal(format string, args ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.errors = append(ml.errors, format)
}

func (ml *MockLogger) LogHTTPRequest(level schemas.LogLevel, msg string) schemas.LogEventBuilder {
	return schemas.NoopLogEvent
}
