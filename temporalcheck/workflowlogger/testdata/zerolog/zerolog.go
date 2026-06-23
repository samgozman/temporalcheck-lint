// Package zerolog is a minimal stand-in for github.com/rs/zerolog. It exists only
// so the analyzer's testdata type-checks without vendoring the real library. The
// analyzer matches it by import path (any package under github.com/rs/zerolog),
// so only the static shape of the chained API -- Logger.Info() returning an
// *Event with a terminal Msg/Send -- needs to be reproduced.
package zerolog

// Logger mirrors zerolog.Logger; its level methods start a logging chain.
type Logger struct{}

// New mirrors zerolog.New(w); fixtures pass os.Stdout.
func New(w interface{}) Logger { return Logger{} }

// Info and Error start a chain, returning the *Event the caller finishes with Msg.
func (l Logger) Info() *Event  { return nil }
func (l Logger) Error() *Event { return nil }

// Event mirrors zerolog.Event, the chained builder a log line is assembled on.
type Event struct{}

// Str adds a field and returns the event for chaining.
func (e *Event) Str(key, val string) *Event { return e }

// Msg, Msgf and Send terminate a chain and emit the line.
func (e *Event) Msg(msg string)                       {}
func (e *Event) Msgf(format string, v ...interface{}) {}
func (e *Event) Send()                                {}
