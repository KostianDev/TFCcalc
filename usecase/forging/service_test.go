package forging

import (
	"testing"
	"tfccalc/domain"
)

func TestSolveForSimpleStamp(t *testing.T) {
	svc := NewService()

	// target 2 can be reached by a single Stamp (delta +2)
	pattern := [3]domain.FinalRequirement{
		{Kind: domain.FinalAny},
		{Kind: domain.FinalAny},
		{Kind: domain.FinalExact, Action: domain.Stamp},
	}
	seq, err := svc.SolveForTarget(2, pattern)
	if err != nil {
		t.Fatalf("expected solution for target=2, got error: %v", err)
	}
	// verify sum
	sum := 0
	for _, a := range seq {
		sum += domain.ForgeActionDelta[a]
	}
	if sum != 2 {
		t.Fatalf("sequence sum = %d; want 2", sum)
	}
	// verify last action is stamp
	if len(seq) == 0 || seq[len(seq)-1] != domain.Stamp {
		t.Fatalf("last action = %v; want Stamp", seq)
	}
}

func TestSolveImpossible(t *testing.T) {
	svc := NewService()
	// With default deltas the gcd is 1 so many targets are reachable.
	// To test failure, constrain maxSteps to force no solution found.
	svc.maxSteps = 1
	pattern := [3]domain.FinalRequirement{{Kind: domain.FinalAny}, {Kind: domain.FinalAny}, {Kind: domain.FinalAny}}
	_, err := svc.SolveForTarget(1, pattern)
	if err == nil {
		t.Fatalf("expected no solution for target=1")
	}
}
