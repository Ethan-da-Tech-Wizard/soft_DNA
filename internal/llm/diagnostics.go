// Boundary-safe summarization loop (BRSL) diagnostics.
//
// LoopState implements the state vector S = (Σ, p, ρ) and update rules from
// KISS_LOOP_EQUATION.md for the per-file LLM summarization loop:
//
//	p ← v·q + (1−v)·p    advance only if the summary validated
//	ρ ← ½ + ¼·v          75% throttle when healthy, 50% after a failure
//
// One diagnostic line is emitted per turn so a run can be watched in real
// time on stderr. The summary memory Σ itself lives with the caller; this
// package only tracks the loop counters.
package llm

import (
	"fmt"
	"io"
	"math"
)

const (
	// RhoNormal is the ingestion throttle while summaries are validating.
	RhoNormal = 0.75
	// RhoRepair is the reduced throttle used to retry after a validation failure.
	RhoRepair = 0.50
	// StallLimit is the maximum consecutive failed turns on one segment before
	// the loop declares an atomic block stall and forces a cutoff summary.
	StallLimit = 2
)

// LoopState tracks the diagnostic variables of the summarization loop.
type LoopState struct {
	Turn  int     // 1-based count of LLM invocations
	T     int     // total segments (files) to process
	P     int     // segments successfully summarized and validated
	Rho   float64 // throttle for the next turn: RhoNormal or RhoRepair
	V     int     // health of the most recent turn: 1 = validated, 0 = failed
	Stall int     // consecutive turns where P failed to advance

	Out io.Writer // destination for diagnostic lines (stderr in production)

	// Captured at BeginTurn so the end-of-turn line reports the coverage and
	// throttle that were in effect when the LLM was invoked.
	turnGamma float64
	turnRho   float64
}

// NewLoopState returns a LoopState ready to track total segments,
// writing diagnostic lines to out.
func NewLoopState(total int, out io.Writer) *LoopState {
	return &LoopState{T: total, Rho: RhoNormal, V: 1, Out: out}
}

// Coverage returns gamma = P/T truncated to two decimals. It is monotone
// non-decreasing across a run: the loop can stall but never lose ground.
func (s *LoopState) Coverage() float64 {
	if s.T == 0 {
		return 0
	}
	return math.Floor(float64(s.P)/float64(s.T)*100) / 100
}

// BeginTurn starts a new turn, capturing the coverage and throttle in effect
// for the imminent LLM invocation.
func (s *LoopState) BeginTurn() {
	s.Turn++
	s.turnGamma = s.Coverage()
	s.turnRho = s.Rho
}

// EndTurn applies the update rules for the turn's validation outcome and
// emits the diagnostic line. On success the pointer advances, the stall
// counter resets, and the throttle returns to normal; on failure the pointer
// freezes, the stall counter increments, and the throttle drops to repair.
func (s *LoopState) EndTurn(ok bool) {
	if ok {
		s.V = 1
		s.P++
		s.Stall = 0
		s.Rho = RhoNormal
	} else {
		s.V = 0
		s.Stall++
		s.Rho = RhoRepair
	}
	fmt.Fprintf(s.Out, "[LLM Loop] Turn %d: Coverage = %.2f | Throttle = %.2f | Health = %d | Stall = %d\n",
		s.Turn, s.turnGamma, s.turnRho, s.V, s.Stall)
}

// Stalled reports whether the current segment has failed more than StallLimit
// consecutive turns — the "atomic block" case where retrying cannot help and
// the caller must force a cutoff summary and move on.
func (s *LoopState) Stalled() bool {
	return s.Stall > StallLimit
}

// ForceAdvance moves the pointer past a stalled segment after a forced cutoff
// summary, restoring the loop to a healthy state so the next segment starts
// at normal throttle.
func (s *LoopState) ForceAdvance() {
	s.V = 1
	s.P++
	s.Stall = 0
	s.Rho = RhoNormal
}
