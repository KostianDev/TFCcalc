package forging

import (
	"fmt"
	"tfccalc/domain"
)

// Service provides forging-related business logic and a solver for anvil sequences.
type Service struct {
	// future: recipe repo, config
	minValue int
	maxValue int
	maxSteps int
}

// NewService constructs a forging service with reasonable defaults.
func NewService() *Service {
	return &Service{
		minValue: 0,
		maxValue: 150,
		maxSteps: 200,
	}
}

// SolveForTarget finds a sequence of actions that reaches `target` starting
// from zero such that the final three actions match `finalPattern`.
// finalPattern must be length 3; use domain.FinalRequirement{Kind: domain.FinalAny}
// for wildcard slots.
func (s *Service) SolveForTarget(target int, finalPattern [3]domain.FinalRequirement) ([]domain.ForgeAction, error) {
	// BFS from 0
	type node struct {
		value int
		seq   []domain.ForgeAction
	}

	// Available actions
	actions := []domain.ForgeAction{
		domain.WeakHit, domain.MediumHit, domain.StrongHit, domain.Draw,
		domain.Stamp, domain.Bend, domain.Upset, domain.Shrink,
	}

	// visited[value] = minimal seq length seen for value
	visited := make(map[int]int)
	q := []node{{value: 0, seq: []domain.ForgeAction{}}}
	visited[0] = 0

	for len(q) > 0 {
		cur := q[0]
		q = q[1:]

		if len(cur.seq) >= s.maxSteps {
			continue
		}

		for _, act := range actions {
			delta := domain.ForgeActionDelta[act]
			newVal := cur.value + delta
			if newVal < s.minValue || newVal > s.maxValue {
				continue
			}
			newSeq := make([]domain.ForgeAction, len(cur.seq)+1)
			copy(newSeq, cur.seq)
			newSeq[len(cur.seq)] = act

			// If we've seen newVal with shorter or equal sequence, skip
			if prevLen, ok := visited[newVal]; ok && prevLen <= len(newSeq) {
				continue
			}
			visited[newVal] = len(newSeq)

			// Check if reached target and suffix matches
			if newVal == target {
				if matchesFinalPattern(newSeq, finalPattern) {
					return newSeq, nil
				}
			}

			q = append(q, node{value: newVal, seq: newSeq})
		}
	}

	return nil, fmt.Errorf("no sequence found to reach %d with given final pattern", target)
}

// matchesFinalPattern returns true if seq's last up to 3 actions match pattern.
func matchesFinalPattern(seq []domain.ForgeAction, pattern [3]domain.FinalRequirement) bool {
	n := len(seq)
	// Align pattern[0..2] with last up to 3 elements of seq
	for i := 0; i < 3; i++ {
		pat := pattern[i]
		seqIndex := n - 3 + i
		if seqIndex < 0 {
			// no corresponding action; skip checks unless pattern requires exact
			if pat.Kind == domain.FinalExact {
				return false
			}
			continue
		}
		act := seq[seqIndex]
		switch pat.Kind {
		case domain.FinalAny:
			continue
		case domain.FinalHit:
			if !domain.IsHit(act) {
				return false
			}
		case domain.FinalExact:
			if act != pat.Action {
				return false
			}
		}
	}
	return true
}
