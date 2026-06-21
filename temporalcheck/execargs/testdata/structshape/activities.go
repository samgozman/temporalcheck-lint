package structshape

import "context"

type Activities struct{}

// TransferParams and EnrichedTransferParams mirror a common case: distinct
// structs that share most fields, differing only at the edges.
type TransferParams struct {
	SourceAccountID string
	Amount          int
	ReferenceID     string
	OnlyOnSource    bool
}

type EnrichedTransferParams struct {
	SourceAccountID string
	Amount          int
	ReferenceID     string
	OnlyOnTarget    *string
}

func (a *Activities) CreateTransfer(ctx context.Context, p *EnrichedTransferParams) error { return nil }

// TargetParams is the wanted shape for the conflict and no-overlap cases.
type TargetParams struct {
	SourceAccountID string
	Amount          int
}

// ConflictParams shares field names with TargetParams but SourceAccountID has an
// incompatible type (int vs string).
type ConflictParams struct {
	SourceAccountID int
	Amount          int
}

func (a *Activities) NeedTarget(ctx context.Context, p *TargetParams) error { return nil }

// Unrelated shares no field names with TargetParams.
type Unrelated struct {
	Foo string
	Bar int
}

// TaggedSend exercises json-tag handling: a renamed field, a skipped field, and
// an unexported field that never reaches the wire.
type TaggedSend struct {
	ID       string `json:"identifier"`
	Secret   string `json:"-"`
	internal int
	Shared   string
}

type TaggedWant struct {
	ID     string `json:"identifier"`
	Shared string
	Extra  string
}

func (a *Activities) NeedTagged(ctx context.Context, p *TaggedWant) error { return nil }

// AParams and BParams have identical fields but are distinct Go types.
type AParams struct {
	X string
	Y int
}

type BParams struct {
	X string
	Y int
}

func (a *Activities) NeedB(ctx context.Context, p *BParams) error { return nil }
