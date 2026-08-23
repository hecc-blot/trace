package mocks

import (
	"context"

	"github.com/hecc-blot/trace/contract"
)

// MockTrace 是 ITrace 接口的 mock 实现，供单测复用。
type MockTrace struct{}

func (m *MockTrace) Start(ctx context.Context, name string, attrs ...interface{}) (context.Context, trace.Span) {
	return ctx, &MockSpan{name: name}
}

func (m *MockTrace) FromContext(ctx context.Context) trace.Span {
	return &MockSpan{}
}

func (m *MockTrace) Inject(ctx context.Context, carrier interface{}) error {
	return nil
}

func (m *MockTrace) Extract(carrier interface{}) (context.Context, error) {
	return context.Background(), nil
}

// MockSpan 是 Span 接口的 mock 实现。
type MockSpan struct {
	name string
}

func (m *MockSpan) End()                                 {}
func (m *MockSpan) SetAttribute(key string, value interface{}) {}
func (m *MockSpan) RecordError(err error)                {}
func (m *MockSpan) Name() string                         { return m.name }
