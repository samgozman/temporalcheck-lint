package bad

import "context"

type Activities struct{}

func (a *Activities) Greet(ctx context.Context, name string) (string, error) {
	return "", nil
}

func (a *Activities) ProcessOrder(ctx context.Context, orderID string, amount int) (bool, error) {
	return false, nil
}

func (a *Activities) Cleanup(jobID string) error {
	return nil
}

func (a *Activities) Notify(ctx context.Context, userID string, tags ...string) error {
	return nil
}

type Payload struct{ ID string }

func (a *Activities) SaveValue(ctx context.Context, p Payload) error { return nil }

func (a *Activities) SavePointer(ctx context.Context, p *Payload) error { return nil }

func (a *Activities) SaveValues(ctx context.Context, ps []Payload) error { return nil }

func (a *Activities) SavePointers(ctx context.Context, ps []*Payload) error { return nil }
