// Package calculator contains the core logic for computing required base materials.
package calculator

import (
	"errors"
	"fmt"
	"log"
	"math"
	"tfccalc/data"
)

// ResolvePercentagesForAlloy gathers and validates a percentage map for the given alloyID.
// If the user provided custom percentages (userPerc), it will be filled out with defaults
// for any missing ingredient, then validated. If userPerc is empty or invalid, defaults are returned.
func ResolvePercentagesForAlloy(alloyID string, userPerc map[string]float64) (map[string]float64, error) {
	alloy, ok := data.GetAlloyByID(alloyID)
	if !ok {
		return nil, fmt.Errorf("alloy %s not found", alloyID)
	}

	// If this alloy has no ingredients, return an empty map
	if len(alloy.Ingredients) == 0 {
		return make(map[string]float64), nil
	}

	// If userPerc is empty, return defaults
	if len(userPerc) == 0 {
		defaults, err := GetDefaultPercentages(alloyID)
		if err != nil {
			return nil, fmt.Errorf("cannot get default percentages for %s: %w", alloyID, err)
		}
		return defaults, nil
	}

	// Copy userPerc so we don't mutate the original
	fullPerc := make(map[string]float64)
	for k, v := range userPerc {
		fullPerc[k] = v
	}

	// If some ingredients are missing, fill with defaults
	if len(fullPerc) < len(alloy.Ingredients) {
		defaults, defErr := GetDefaultPercentages(alloyID)
		if defErr == nil {
			for _, ing := range alloy.Ingredients {
				if _, exists := fullPerc[ing.IngredientID]; !exists {
					fullPerc[ing.IngredientID] = defaults[ing.IngredientID]
				}
			}
		}
	}

	// Validate the completed map of percentages
	valid, valErr := ValidatePercentages(alloyID, fullPerc)
	if valid {
		return fullPerc, nil
	}

	// If user percentages are invalid, log a warning and return defaults
	log.Printf("Warning: invalid user percentages for %s (%v), using defaults", alloyID, valErr)
	defaults, err := GetDefaultPercentages(alloyID)
	if err != nil {
		return nil, fmt.Errorf("cannot get default percentages for %s after invalid user input: %w", alloyID, err)
	}
	return defaults, nil
}

// GetDefaultPercentages computes midpoint percentages between Min and Max
// and ensures they sum to exactly 100. If rounding causes a small discrepancy,
// the difference is added to the first ingredient.
func GetDefaultPercentages(alloyID string) (map[string]float64, error) {
	alloy, ok := data.GetAlloyByID(alloyID)
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

	// If due to rounding, total != 100, adjust the first ingredient
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

// ValidatePercentages checks that:
// 1) all ingredients are present,
// 2) each percentage is within [Min - ε, Max + ε],
// 3) the sum of all percentages is approximately 100.
func ValidatePercentages(alloyID string, percentages map[string]float64) (bool, error) {
	alloy, ok := data.GetAlloyByID(alloyID)
	if !ok {
		return false, fmt.Errorf("alloy %s not found for validation", alloyID)
	}
	// If no ingredients exist, only an empty map is valid
	if len(alloy.Ingredients) == 0 {
		return len(percentages) == 0, nil
	}
	// If the map is empty, treat it as "use defaults"
	if len(percentages) == 0 {
		return true, nil
	}

	// Must have exactly as many keys as there are ingredients
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
			name := data.GetAlloyNameByID(ingData.IngredientID)
			return false, fmt.Errorf("percentage for %s (%.2f%%) outside [%.2f–%.2f] for %s", name, pct, ingData.Min, ingData.Max, alloy.Name)
		}
		total += pct
	}
	if math.Abs(total-100.0) > 0.01 {
		return false, fmt.Errorf("sum of percentages for %s is %.2f%% (should be 100%%)", alloy.Name, total)
	}
	return true, nil
}

// sumMaterials merges two maps of {baseID → amountMB}, adding the values.
func sumMaterials(m1, m2 map[string]float64) map[string]float64 {
	res := make(map[string]float64)
	for k, v := range m1 {
		res[k] = v
	}
	for k, v := range m2 {
		res[k] += v
	}
	return res
}

// getBaseMaterialBreakdown recursively expands the given targetID (any alloy or base)
// into its constituent base materials (type "base"), applying percentages from allUserPerc.
func getBaseMaterialBreakdown(targetID string, amountMB float64, allUserPerc map[string]map[string]float64, level int) (map[string]float64, error) {
	if level > 20 {
		return nil, errors.New("maximum recursion depth exceeded, possible cyclic dependency")
	}
	targetData, ok := data.GetAlloyByID(targetID)
	if !ok {
		return nil, fmt.Errorf("unknown material ID %s", targetID)
	}

	// If it's a base material, return directly
	if targetData.Type == "base" {
		return map[string]float64{targetID: amountMB}, nil
	}

	// If it's plain "Steel", resolve to pig_iron at 100%
	if targetID == "steel" {
		return getBaseMaterialBreakdown("pig_iron", amountMB, allUserPerc, level+1)
	}

	// If it's a final steel (e.g. "black_steel"), process RawForm + ExtraIngredient
	if targetData.Type == "final_steel" {
		if !targetData.RawFormID.Valid || !targetData.ExtraIngredientID.Valid {
			return nil, fmt.Errorf("incomplete data for final_steel %s", targetID)
		}
		// First: break down the raw form
		rawCost, err := getBaseMaterialBreakdown(targetData.RawFormID.String, amountMB, allUserPerc, level+1)
		if err != nil {
			return nil, fmt.Errorf("error calculating rawForm for %s: %w", targetID, err)
		}
		// Second: break down the extra ingredient (pig_iron or another steel)
		extraCost, err := getBaseMaterialBreakdown(targetData.ExtraIngredientID.String, amountMB, allUserPerc, level+1)
		if err != nil {
			return nil, fmt.Errorf("error calculating extraIngredient for %s: %w", targetID, err)
		}
		// Merge both maps and return
		return sumMaterials(rawCost, extraCost), nil
	}

	// If it's an intermediate alloy, raw_steel, or processed (other than "steel")
	if targetData.Type == "alloy" || targetData.Type == "raw_steel" || targetData.Type == "processed" {
		// If there are no ingredients, return an empty map
		if len(targetData.Ingredients) == 0 {
			return make(map[string]float64), nil
		}
		// Determine which percentages to use (resolve with user overrides or defaults)
		var percentagesToUse map[string]float64
		if userMap, found := allUserPerc[targetID]; found {
			resolved, err := ResolvePercentagesForAlloy(targetID, userMap)
			if err != nil {
				// Log warning, but fall back to defaults
				log.Printf("Warning: cannot resolve user percentages for %s: %v, using defaults", targetID, err)
				defaults, _ := GetDefaultPercentages(targetID)
				percentagesToUse = defaults
			} else {
				percentagesToUse = resolved
			}
		} else {
			defaults, err := GetDefaultPercentages(targetID)
			if err != nil {
				return nil, fmt.Errorf("cannot get default percentages for %s: %w", targetData.Name, err)
			}
			percentagesToUse = defaults
		}

		// Recursively break down each ingredient
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
			sub, err := getBaseMaterialBreakdown(ing.IngredientID, requiredMB, allUserPerc, level+1)
			if err != nil {
				return nil, fmt.Errorf("error expanding %s for %s: %w", ing.IngredientID, targetID, err)
			}
			total = sumMaterials(total, sub)
		}
		return total, nil
	}

	return nil, fmt.Errorf("unhandled material type %s for %s", targetData.Type, targetID)
}

// CalculateRequirements is the main function called by UI.
// - targetID: ID of the alloy or steel (e.g. "blue_steel", "brass", etc.)
// - amount: quantity (in MB or ingots, depending on mode)
// - mode: "mB" or "Ingots"
// - allUserPerc: nested map[alloyID] → (map[ingredientID] → pct) with any user overrides.
// Returns two maps: {baseID → mB} and {baseID → Ingots}, or an error.
func CalculateRequirements(
	targetID string,
	amount float64,
	mode string,
	allUserPerc map[string]map[string]float64,
) (map[string]float64, map[string]float64, error) {
	// --- Input validation ---
	if amount <= 0 {
		return nil, nil, errors.New("amount must be positive")
	}
	if mode != "mB" && mode != "Ingots" {
		return nil, nil, errors.New("invalid mode; only \"mB\" or \"Ingots\"")
	}
	targetData, ok := data.GetAlloyByID(targetID)
	if !ok {
		return nil, nil, fmt.Errorf("alloy %s not found", targetID)
	}

	// --- Top‐level percentage validation (if user provided overrides for this level) ---
	idForValidation := targetID
	if targetData.Type == "final_steel" {
		// For final steel, validate on its RawForm
		idForValidation = targetData.RawFormID.String
	}
	if userMap, found := allUserPerc[idForValidation]; found && len(userMap) > 0 {
		_, err := ResolvePercentagesForAlloy(idForValidation, userMap)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid user percentages for %s: %w", data.GetAlloyNameByID(idForValidation), err)
		}
		// Replace user map with the fully resolved one (including defaults)
		allUserPerc[idForValidation], _ = ResolvePercentagesForAlloy(idForValidation, userMap)
	}

	// --- Convert to mB if in "Ingots" mode ---
	var amountMB float64
	if mode == "Ingots" {
		amountMB = amount * 100.0
	} else {
		amountMB = amount
	}

	finalMaterialsMB := make(map[string]float64)

	// Handle final steels separately (RawForm + ExtraIngredient)
	if targetData.Type == "final_steel" {
		raw, err := getBaseMaterialBreakdown(targetData.RawFormID.String, amountMB, allUserPerc, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("error calculating raw form for %s: %w", targetID, err)
		}
		extra, err := getBaseMaterialBreakdown(targetData.ExtraIngredientID.String, amountMB, allUserPerc, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("error calculating extra ingredient for %s: %w", targetID, err)
		}
		finalMaterialsMB = sumMaterials(raw, extra)
	} else {
		// Non‐final materials: break down directly
		need, err := getBaseMaterialBreakdown(targetID, amountMB, allUserPerc, 0)
		if err != nil {
			return nil, nil, err
		}
		finalMaterialsMB = need
	}

	// Build the {baseID → Ingots} map
	finalMaterialsIngots := make(map[string]float64)
	for id, mB := range finalMaterialsMB {
		finalMaterialsIngots[id] = mB / 100.0
	}

	// Edge case: if nothing returned (e.g. base material in "mB" mode), treat it as itself
	if len(finalMaterialsMB) == 0 && targetData.Type == "base" && mode == "mB" {
		finalMaterialsMB[targetID] = amountMB
		finalMaterialsIngots[targetID] = amountMB / 100.0
	}

	return finalMaterialsMB, finalMaterialsIngots, nil
}
