// Package log is a minimal stand-in for github.com/rs/zerolog/log, zerolog's
// global-logger convenience package. The analyzer matches it by import path
// (it is under github.com/rs/zerolog), so a chain like log.Info().Msg("x") is
// flagged the same as a method on a Logger value.
package log

import "github.com/rs/zerolog"

// Logger is the package-level logger the convenience helpers delegate to.
var Logger = zerolog.Logger{}

// Info and Error mirror the global helpers, starting a chain off the package logger.
func Info() *zerolog.Event  { return Logger.Info() }
func Error() *zerolog.Event { return Logger.Error() }
