package pipeline

import (
	"context"
	"log/slog"
	"time"
)

// Event represents a pipeline event with context and metadata
type Event struct {
	Type      EventType      `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	StageName string         `json:"stage_name,omitempty"`
	WorkerID  int            `json:"worker_id,omitempty"`
	Message   string         `json:"message,omitempty"`
	Error     error          `json:"error,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Observer interface for observing pipeline events
type Observer interface {
	OnEvent(ctx context.Context, event Event)
}

// Logger interface wraps slog.Logger for pipeline observability
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// SlogLogger wraps slog.Logger to implement our Logger interface
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a Logger that wraps slog.Logger
func NewSlogLogger(logger *slog.Logger) Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogLogger{logger: logger}
}

func (s *SlogLogger) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *SlogLogger) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *SlogLogger) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *SlogLogger) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

func (s *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{logger: s.logger.With(args...)}
}

// ObserverFunc is a function adapter for Observer interface
type ObserverFunc func(ctx context.Context, event Event)

func (f ObserverFunc) OnEvent(ctx context.Context, event Event) {
	f(ctx, event)
}

// MultiObserver combines multiple observers
type MultiObserver struct {
	observers []Observer
}

// NewMultiObserver creates a new MultiObserver
func NewMultiObserver(observers ...Observer) *MultiObserver {
	return &MultiObserver{observers: observers}
}

// OnEvent broadcasts the event to all observers
func (m *MultiObserver) OnEvent(ctx context.Context, event Event) {
	for _, observer := range m.observers {
		observer.OnEvent(ctx, event)
	}
}

// Add adds an observer to the MultiObserver
func (m *MultiObserver) Add(observer Observer) {
	m.observers = append(m.observers, observer)
}

// NoOpObserver is a no-operation observer (default)
type NoOpObserver struct{}

func (NoOpObserver) OnEvent(ctx context.Context, event Event) {}

// NoOpLogger is a no-operation logger (default)
type NoOpLogger struct{}

func (NoOpLogger) Debug(msg string, args ...any) {}
func (NoOpLogger) Info(msg string, args ...any)  {}
func (NoOpLogger) Warn(msg string, args ...any)  {}
func (NoOpLogger) Error(msg string, args ...any) {}
func (NoOpLogger) With(args ...any) Logger       { return NoOpLogger{} }

// LoggerObserver adapts a Logger to Observer interface
type LoggerObserver struct {
	logger Logger
}

// NewLoggerObserver creates a new LoggerObserver
func NewLoggerObserver(logger Logger) *LoggerObserver {
	return &LoggerObserver{logger: logger}
}

// OnEvent handles events by logging them with appropriate levels
func (l *LoggerObserver) OnEvent(ctx context.Context, event Event) {
	args := []any{
		"type", event.Type,
		"timestamp", event.Timestamp,
	}

	if event.StageName != "" {
		args = append(args, "stage", event.StageName)
	}
	if event.WorkerID != 0 {
		args = append(args, "worker_id", event.WorkerID)
	}
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			args = append(args, k, v)
		}
	}

	switch event.Type {
	case EventError, EventItemFailed, EventRouteError:
		if event.Error != nil {
			args = append(args, "error", event.Error)
		}
		l.logger.Error(event.Message, args...)
	case EventWarning, EventUnsupportedType, EventNoMatch:
		l.logger.Warn(event.Message, args...)
	case EventDebug:
		l.logger.Debug(event.Message, args...)
	default:
		l.logger.Info(event.Message, args...)
	}
}

// PipelineConfig holds configuration for pipeline observability
type PipelineConfig struct {
	Observer Observer
	Logger   Logger
	Context  context.Context
}

// DefaultConfig returns a default pipeline configuration with no-op implementations
func DefaultConfig() *PipelineConfig {
	return &PipelineConfig{
		Observer: NoOpObserver{},
		Logger:   NewSlogLogger(slog.Default()),
		Context:  context.Background(),
	}
}

// DefaultConfigWithSlog creates a config with the provided slog.Logger
func DefaultConfigWithSlog(logger *slog.Logger) *PipelineConfig {
	return &PipelineConfig{
		Observer: NoOpObserver{},
		Logger:   NewSlogLogger(logger),
		Context:  context.Background(),
	}
}

// WithObserver sets the observer in the config
func (c *PipelineConfig) WithObserver(observer Observer) *PipelineConfig {
	c.Observer = observer
	return c
}

// WithLogger sets the logger in the config
func (c *PipelineConfig) WithLogger(logger Logger) *PipelineConfig {
	c.Logger = logger
	return c
}

// WithContext sets the context in the config
func (c *PipelineConfig) WithContext(ctx context.Context) *PipelineConfig {
	c.Context = ctx
	return c
}

// EmitEvent is a helper function to emit events safely
func EmitEvent(config *PipelineConfig, eventType EventType, stageName, message string, err error, metadata map[string]interface{}) {
	if config == nil || config.Observer == nil {
		return
	}

	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		StageName: stageName,
		Message:   message,
		Error:     err,
		Metadata:  metadata,
	}

	config.Observer.OnEvent(config.Context, event)
}

// EmitStageEvent is a helper for stage-specific event
func EmitStageEvent(config *PipelineConfig, eventType EventType, stageName string, workerID int, message string, err error) {
	if config == nil || config.Observer == nil {
		return
	}

	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		StageName: stageName,
		WorkerID:  workerID,
		Message:   message,
		Error:     err,
	}

	config.Observer.OnEvent(config.Context, event)
}
