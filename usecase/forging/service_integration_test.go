package forging

import (
	"testing"
	"tfccalc/domain"
)

// TestSolveTarget82Hits ensures solver can find a sequence to reach 82 with
// final three actions being any Hits.
func TestSolveTarget82Hits(t *testing.T) {
	svc := NewService()
	// First, try any-final pattern to see if target is reachable at all.
	anyPattern := [3]domain.FinalRequirement{{Kind: domain.FinalAny}, {Kind: domain.FinalAny}, {Kind: domain.FinalAny}}
	seqAny, errAny := svc.SolveForTarget(82, anyPattern)
	if errAny != nil {
		t.Fatalf("expected some sequence to reach 82 (any-final), got error: %v", errAny)
	}
	t.Logf("found sequence (any-final) len=%d: %v", len(seqAny), seqAny)

	pattern := [3]domain.FinalRequirement{{Kind: domain.FinalHit}, {Kind: domain.FinalHit}, {Kind: domain.FinalHit}}

	seq, err := svc.SolveForTarget(82, pattern)
	if err != nil {
		t.Fatalf("expected solution for 82 with Hit-Hit-Hit, got error: %v", err)
	}
	if len(seq) < 3 {
		t.Fatalf("sequence too short: %v", seq)
	}

	// Check last three are hits and final value computes to 82.
	for i := len(seq) - 3; i < len(seq); i++ {
		if !domain.IsHit(seq[i]) {
			t.Fatalf("expected hit in final three, got %v", seq[i])
		}
	}

	// Compute final value
	val := 0
	for _, a := range seq {
		val += domain.ForgeActionDelta[a]
	}
	if val != 82 {
		t.Fatalf("sequence does not reach 82, reached %d, seq=%v", val, seq)
	}
}

// Sanity check that a handcrafted sequence of 7 Upset then 3 WeakHit reaches 82
func TestHandcraftedSequence(t *testing.T) {
	seq := []domain.ForgeAction{}
	for i := 0; i < 7; i++ {
		seq = append(seq, domain.Upset)
	}
	for i := 0; i < 3; i++ {
		seq = append(seq, domain.WeakHit)
	}

	// Compute final value
	val := 0
	for _, a := range seq {
		val += domain.ForgeActionDelta[a]
	}
	if val != 82 {
		t.Fatalf("handcrafted sequence did not reach 82, got %d", val)
	}

	// Check final-three match
	pattern := [3]domain.FinalRequirement{{Kind: domain.FinalHit}, {Kind: domain.FinalHit}, {Kind: domain.FinalHit}}
	if !matchesFinalPattern(seq, pattern) {
		t.Fatalf("handcrafted sequence should match final Hit-Hit-Hit pattern: %v", seq)
	}
}
