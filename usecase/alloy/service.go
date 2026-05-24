package alloy

import (
	"errors"
	"fmt"
	"log"
	"math"
	"tfccalc/domain"
)

// Service orchestrates alloy calculations and percentage validation.
type Service struct {
	repo domain.AlloyRepository
}

// NewService creates a new alloy service.
func NewService(repo domain.AlloyRepository) *Service {
	return &Service{repo: repo}
}

// GetAlloyByID returns an alloy by ID.
func (s *Service) GetAlloyByID(id string) (domain.Alloy, bool) {
	return s.repo.GetAlloyByID(id)
}

// GetAllAlloys returns all alloys.
func (s *Service) GetAllAlloys() map[string]domain.Alloy {
	return s.repo.GetAllAlloys()
}

// GetAlloyNameByID returns the name of an alloy by ID.
func (s *Service) GetAlloyNameByID(id string) string {
	return s.repo.GetAlloyNameByID(id)
}

// ResolvePercentagesForAlloy builds and validates percentages for an alloy.
func (s *Service) ResolvePercentagesForAlloy(alloyID string, userPerc map[string]float64) (map[string]float64, error) {
	alloy, ok := s.repo.GetAlloyByID(alloyID)
	if !ok {
		return nil, fmt.Errorf("alloy %s not found", alloyID)
	}

	if len(alloy.Ingredients) == 0 {
		return make(map[string]float64), nil
	}

	if len(userPerc) == 0 {
		defaults, err := s.GetDefaultPercentages(alloyID)
		if err != nil {
			return nil, fmt.Errorf("cannot get default percentages for %s: %w", alloyID, err)
		}
		return defaults, nil
	}

	fullPerc := make(map[string]float64)
	for k, v := range userPerc {
		fullPerc[k] = v
	}

	if len(fullPerc) < len(alloy.Ingredients) {
		defaults, defErr := s.GetDefaultPercentages(alloyID)
		if defErr == nil {
			for _, ing := range alloy.Ingredients {
				if _, exists := fullPerc[ing.IngredientID]; !exists {
					fullPerc[ing.IngredientID] = defaults[ing.IngredientID]
				}
			}
		}
	}

	valid, valErr := s.ValidatePercentages(alloyID, fullPerc)
	if valid {
		return fullPerc, nil
	}

	log.Printf("Warning: invalid user percentages for %s (%v), using defaults", alloyID, valErr)
	defaults, err := s.GetDefaultPercentages(alloyID)
	if err != nil {
		return nil, fmt.Errorf("cannot get default percentages for %s after invalid user input: %w", alloyID, err)
	}
	return defaults, nil
}

// GetDefaultPercentages computes midpoints and ensures the sum is 100.
func (s *Service) GetDefaultPercentages(alloyID string) (map[string]float64, error) {
	alloy, ok := s.repo.GetAlloyByID(alloyID)
	if !ok {
		return nil, fmt.Errorf("alloy %s not found", alloyID)
	}
	if len(alloy.Ingredients) == 0 {
		return make(map[string]float64), nil
	}

	percentages := make(map[string]float64)
	total := 0.0
	for _, ing := range alloy.Ingredients {
		mid := (ing.Min + ing.Max) / 2.0
		percentages[ing.IngredientID] = mid
		total += mid
	}

	if math.Abs(total-100.0) > 0.01 && len(alloy.Ingredients) > 0 {
		diff := 100.0 - total
		firstID := alloy.Ingredients[0].IngredientID
		if _, exists := percentages[firstID]; exists {
			percentages[firstID] += diff
		} else {
			return nil, fmt.Errorf("internal error: ingredient %s missing when adjusting defaults for %s", firstID, alloyID)
		}
	}
	return percentages, nil
}

// ValidatePercentages checks presence, bounds, and sum near 100.
func (s *Service) ValidatePercentages(alloyID string, percentages map[string]float64) (bool, error) {
	alloy, ok := s.repo.GetAlloyByID(alloyID)
	if !ok {
		return false, fmt.Errorf("alloy %s not found for validation", alloyID)
	}
	if len(alloy.Ingredients) == 0 {
		return len(percentages) == 0, nil
	}
	if len(percentages) == 0 {
		return true, nil
	}
	if len(percentages) != len(alloy.Ingredients) {
		return false, fmt.Errorf("expected %d ingredients for %s, got %d", len(alloy.Ingredients), alloyID, len(percentages))
	}

	total := 0.0
	eps := 0.001
	for _, ingData := range alloy.Ingredients {
		pct, found := percentages[ingData.IngredientID]
		if !found {
			return false, fmt.Errorf("percentage for %s missing in map for %s", ingData.IngredientID, alloyID)
		}
		if pct < ingData.Min-eps || pct > ingData.Max+eps {
			name := s.repo.GetAlloyNameByID(ingData.IngredientID)
			return false, fmt.Errorf("percentage for %s (%.2f%%) outside [%.2f-%.2f] for %s", name, pct, ingData.Min, ingData.Max, alloy.Name)
		}
		total += pct
	}
	if math.Abs(total-100.0) > 0.01 {
		return false, fmt.Errorf("sum of percentages for %s is %.2f%% (should be 100%%)", alloy.Name, total)
	}
	return true, nil
}

// CalculateRequirements computes base material requirements for an alloy.
func (s *Service) CalculateRequirements(
	targetID string,
	amount float64,
	mode string,
	allUserPerc map[string]map[string]float64,
) (map[string]float64, map[string]float64, error) {
	if amount <= 0 {
		return nil, nil, errors.New("amount must be positive")
	}
	if mode != "mB" && mode != "Ingots" {
		return nil, nil, errors.New("invalid mode; only \"mB\" or \"Ingots\"")
	}
	if _, ok := s.repo.GetAlloyByID(targetID); !ok {
		return nil, nil, fmt.Errorf("alloy %s not found", targetID)
	}

	amountMB := amount
	if mode == "Ingots" {
		amountMB = amount * 100.0
	}

	finalMaterialsMB, err := s.getBaseMaterialBreakdown(targetID, amountMB, allUserPerc, 0)
	if err != nil {
		return nil, nil, err
	}

	finalMaterialsIngots := make(map[string]float64)
	for id, mb := range finalMaterialsMB {
		finalMaterialsIngots[id] = mb / 100.0
	}

	return finalMaterialsMB, finalMaterialsIngots, nil
}

func (s *Service) sumMaterials(m1, m2 map[string]float64) map[string]float64 {
	res := make(map[string]float64)
	for k, v := range m1 {
		res[k] = v
	}
	for k, v := range m2 {
		res[k] += v
	}
	return res
}

func (s *Service) getBaseMaterialBreakdown(
	targetID string,
	amountMB float64,
	allUserPerc map[string]map[string]float64,
	level int,
) (map[string]float64, error) {
	if level > 20 {
		return nil, errors.New("maximum recursion depth exceeded, possible cyclic dependency")
	}
	targetData, ok := s.repo.GetAlloyByID(targetID)
	if !ok {
		return nil, fmt.Errorf("unknown material ID %s", targetID)
	}

	if targetData.Type == domain.AlloyTypeBase {
		return map[string]float64{targetID: amountMB}, nil
	}

	if targetID == "steel" {
		return s.getBaseMaterialBreakdown("pig_iron", amountMB, allUserPerc, level+1)
	}

	if targetData.Type == domain.AlloyTypeFinalSteel {
		if targetData.RawFormID == nil || targetData.ExtraIngredientID == nil {
			return nil, fmt.Errorf("incomplete data for final_steel %s", targetID)
		}
		rawCost, err := s.getBaseMaterialBreakdown(*targetData.RawFormID, amountMB, allUserPerc, level+1)
		if err != nil {
			return nil, fmt.Errorf("error calculating rawForm for %s: %w", targetID, err)
		}
		extraCost, err := s.getBaseMaterialBreakdown(*targetData.ExtraIngredientID, amountMB, allUserPerc, level+1)
		if err != nil {
			return nil, fmt.Errorf("error calculating extraIngredient for %s: %w", targetID, err)
		}
		return s.sumMaterials(rawCost, extraCost), nil
	}

	if targetData.Type == domain.AlloyTypeAlloy ||
		targetData.Type == domain.AlloyTypeRawSteel ||
		targetData.Type == domain.AlloyTypeProcessed {
		if len(targetData.Ingredients) == 0 {
			return make(map[string]float64), nil
		}

		var percentagesToUse map[string]float64
		if userMap, found := allUserPerc[targetID]; found {
			resolved, err := s.ResolvePercentagesForAlloy(targetID, userMap)
			if err != nil {
				log.Printf("Warning: cannot resolve user percentages for %s: %v, using defaults", targetID, err)
				defaults, _ := s.GetDefaultPercentages(targetID)
				percentagesToUse = defaults
			} else {
				percentagesToUse = resolved
			}
		} else {
			defaults, err := s.GetDefaultPercentages(targetID)
			if err != nil {
				return nil, fmt.Errorf("cannot get default percentages for %s: %w", targetData.Name, err)
			}
			percentagesToUse = defaults
		}

		total := make(map[string]float64)
		for _, ing := range targetData.Ingredients {
			pct, exists := percentagesToUse[ing.IngredientID]
			if !exists {
				return nil, fmt.Errorf("internal error: ingredient %s missing after resolving for %s", ing.IngredientID, targetID)
			}
			requiredMB := amountMB * (pct / 100.0)
			if requiredMB < 0.001 {
				continue
			}
			sub, err := s.getBaseMaterialBreakdown(ing.IngredientID, requiredMB, allUserPerc, level+1)
			if err != nil {
				return nil, fmt.Errorf("error expanding %s for %s: %w", ing.IngredientID, targetID, err)
			}
			total = s.sumMaterials(total, sub)
		}
		return total, nil
	}

	return nil, fmt.Errorf("unhandled material type %s for %s", targetData.Type, targetID)
}
