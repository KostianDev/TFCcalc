package forging

import (
	"container/heap"
	"fmt"
	"math"
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

// priority queue item and heap implementation at package scope
type pqItem struct {
	priority int // f = g + h
	g        int // actual cost so far (steps)
	value    int
	seq      []domain.ForgeAction
	index    int // heap index
}

// pq is a min-heap of *pqItem
type pq []*pqItem

func (p pq) Len() int           { return len(p) }
func (p pq) Less(i, j int) bool { return p[i].priority < p[j].priority }
func (p pq) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].index = i
	p[j].index = j
}
func (p *pq) Push(x interface{}) {
	it := x.(*pqItem)
	it.index = len(*p)
	*p = append(*p, it)
}
func (p *pq) Pop() interface{} {
	old := *p
	n := len(old)
	it := old[n-1]
	it.index = -1
	*p = old[0 : n-1]
	return it
}

// compact action codes (0 reserved for empty slot)
var actionCode = map[domain.ForgeAction]int{
	domain.WeakHit:   1,
	domain.MediumHit: 2,
	domain.StrongHit: 3,
	domain.Draw:      4,
	domain.Stamp:     5,
	domain.Bend:      6,
	domain.Upset:     7,
	domain.Shrink:    8,
}

// SolveForTarget finds a sequence of actions that reaches `target` starting
// from zero such that the final three actions match `finalPattern`.
// finalPattern must be length 3; use domain.FinalRequirement{Kind: domain.FinalAny}
// for wildcard slots.
func (s *Service) SolveForTarget(target int, finalPattern [3]domain.FinalRequirement) ([]domain.ForgeAction, error) {
	// A* search from 0 where cost g = number of steps and heuristic h =
	// minimal remaining steps based on largest single-action effect.

	// Available actions
	actions := []domain.ForgeAction{
		domain.WeakHit, domain.MediumHit, domain.StrongHit, domain.Draw,
		domain.Stamp, domain.Bend, domain.Upset, domain.Shrink,
	}

	// Precompute max positive and max negative absolute effects for heuristic.
	maxPos := 0
	maxNegAbs := 0
	for _, d := range domain.ForgeActionDelta {
		if d > 0 && d > maxPos {
			maxPos = d
		}
		if d < 0 && -d > maxNegAbs {
			maxNegAbs = -d
		}
	}
	if maxPos == 0 {
		maxPos = 1
	}
	if maxNegAbs == 0 {
		maxNegAbs = 1
	}

	// heuristic: admissible estimate of remaining steps
	h := func(val int) int {
		if val == target {
			return 0
		}
		diff := target - val
		if diff > 0 {
			return int(math.Ceil(float64(diff) / float64(maxPos)))
		}
		return int(math.Ceil(float64(-diff) / float64(maxNegAbs)))
	}

	// key: compact int: (value << 12) | tailCode (3 actions × 4 bits)
	visited := make(map[int]int)

	// initial state key (value=0, empty tail)
	visited[0] = 0
	start := &pqItem{priority: h(0), g: 0, value: 0, seq: []domain.ForgeAction{}}
	open := pq{start}
	heap.Init(&open)

	for open.Len() > 0 {
		curIt := heap.Pop(&open).(*pqItem)
		// prune by maxSteps
		if curIt.g >= s.maxSteps {
			continue
		}

		// If at target and suffix matches, return solution.
		if curIt.value == target {
			if matchesFinalPattern(curIt.seq, finalPattern) {
				return curIt.seq, nil
			}
			// otherwise continue exploring different suffixes.
		}

		// expand
		for _, act := range actions {
			delta := domain.ForgeActionDelta[act]
			newVal := curIt.value + delta
			if newVal < s.minValue || newVal > s.maxValue {
				continue
			}
			newSeq := make([]domain.ForgeAction, len(curIt.seq)+1)
			copy(newSeq, curIt.seq)
			newSeq[len(curIt.seq)] = act

			// build compact tail code: 3 slots × 4 bits (0 reserved for empty)
			tailStart := 0
			if len(newSeq) > 3 {
				tailStart = len(newSeq) - 3
			}
			tailCode := 0
			for i := tailStart; i < len(newSeq); i++ {
				code := actionCode[newSeq[i]]
				tailCode = (tailCode << 4) | (code & 0xF)
			}
			key := (newVal << 12) | tailCode

			tentativeG := curIt.g + 1
			if prevG, ok := visited[key]; ok && prevG <= tentativeG {
				continue
			}
			visited[key] = tentativeG

			f := tentativeG + h(newVal)
			heap.Push(&open, &pqItem{priority: f, g: tentativeG, value: newVal, seq: newSeq})
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
