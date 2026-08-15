package trace

import (
	"context"
)

type ITrace interface {
	Start(ctx context.Context, name string, attrs ...interface{}) (context.Context, Span)
	FromContext(ctx context.Context) Span
	Inject(ctx context.Context, carrier interface{}) error
	Extract(carrier interface{}) (context.Context, error)
}

type Span interface {
	End()
	SetAttribute(key string, value interface{})
	RecordError(err error)
	Name() string
}
