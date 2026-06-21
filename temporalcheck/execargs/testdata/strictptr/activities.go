package strictptr

import "context"

type Activities struct{}

type Payload struct{ ID string }

func (a *Activities) SaveValue(ctx context.Context, p Payload) error { return nil }

func (a *Activities) SavePointer(ctx context.Context, p *Payload) error { return nil }

func (a *Activities) SaveValues(ctx context.Context, ps []Payload) error { return nil }

func (a *Activities) SavePointers(ctx context.Context, ps []*Payload) error { return nil }
