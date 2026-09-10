package core

import "context"

type Rules interface {
	Evaluate(context.Context, RouteRequest) (ZenDecision, error)
}
type Registrations interface {
	Current() []Registration
}
type Decisions interface {
	Find(context.Context, string) (string, []byte, bool, error)
	Save(context.Context, string, string, []byte) ([]byte, bool, error)
	Binding(context.Context, string, string, string, string, string) (*ProviderBinding, error)
}
