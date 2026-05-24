package calculator

import (
	"errors"
	"tfccalc/data"
	"tfccalc/usecase/alloy"
)

func getService() (*alloy.Service, error) {
	repo := data.Repository()
	if repo == nil {
		return nil, errors.New("repository not initialized")
	}
	return alloy.NewService(repo), nil
}

// ResolvePercentagesForAlloy builds and validates percentages for an alloy.
func ResolvePercentagesForAlloy(alloyID string, userPerc map[string]float64) (map[string]float64, error) {
	svc, err := getService()
	if err != nil {
		return nil, err
	}
	return svc.ResolvePercentagesForAlloy(alloyID, userPerc)
}

// GetDefaultPercentages computes midpoints and ensures the sum is 100.
func GetDefaultPercentages(alloyID string) (map[string]float64, error) {
	svc, err := getService()
	if err != nil {
		return nil, err
	}
	return svc.GetDefaultPercentages(alloyID)
}

// ValidatePercentages checks presence, bounds, and sum near 100.
func ValidatePercentages(alloyID string, percentages map[string]float64) (bool, error) {
	svc, err := getService()
	if err != nil {
		return false, err
	}
	return svc.ValidatePercentages(alloyID, percentages)
}

// CalculateRequirements computes base material requirements for an alloy.
func CalculateRequirements(
	targetID string,
	amount float64,
	mode string,
	allUserPerc map[string]map[string]float64,
) (map[string]float64, map[string]float64, error) {
	svc, err := getService()
	if err != nil {
		return nil, nil, err
	}
	return svc.CalculateRequirements(targetID, amount, mode, allUserPerc)
}
