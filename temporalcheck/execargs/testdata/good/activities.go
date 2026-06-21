package good

import "context"

// Activities groups the example activity implementations. In real code these
// hang off a struct so they can capture dependencies (clients, config, ...).
type Activities struct{}

// Greet takes the injected context plus one real argument.
func (a *Activities) Greet(ctx context.Context, name string) (string, error) {
	return "hello " + name, nil
}

// ProcessOrder takes two real arguments after the context.
func (a *Activities) ProcessOrder(ctx context.Context, orderID string, amount int) (bool, error) {
	return true, nil
}

// Cleanup omits the optional leading context.Context.
func (a *Activities) Cleanup(jobID string) error {
	return nil
}

// Notify is a variadic activity: a fixed userID plus zero or more tags.
func (a *Activities) Notify(ctx context.Context, userID string, tags ...string) error {
	return nil
}

// ArchiveAll is a package-level function activity (no receiver).
func ArchiveAll(ctx context.Context, bucket string) error {
	return nil
}

// Payload is a struct argument used to exercise value-vs-pointer and
// slice-element-pointer assignability.
type Payload struct{ ID string }

// SaveValue takes a struct value.
func (a *Activities) SaveValue(ctx context.Context, p Payload) error { return nil }

// SavePointer takes a pointer to the struct.
func (a *Activities) SavePointer(ctx context.Context, p *Payload) error { return nil }

// SaveValues takes a slice of struct values.
func (a *Activities) SaveValues(ctx context.Context, ps []Payload) error { return nil }

// SavePointers takes a slice of pointers.
func (a *Activities) SavePointers(ctx context.Context, ps []*Payload) error { return nil }
