package usecase

import "tfccalc/domain"

// AlloyCalculator calculates material requirements for a target alloy.
type AlloyCalculator interface {
	CalculateRequirements(
		targetID string,
		amount float64,
		mode string,
		userPerc map[string]map[string]float64,
	) (map[string]float64, map[string]float64, error)
}

// AnvilSolver computes a sequence of actions to reach a target value
// while satisfying the last-actions constraint.
type AnvilSolver interface {
	Solve(target int, lastActions []string) ([]string, error)
}

// RepositoryProvider exposes the repositories required by use cases.
type RepositoryProvider interface {
	Alloys() domain.AlloyRepository
}
