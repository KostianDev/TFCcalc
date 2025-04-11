// Package calculator contains the core logic for calculating alloy component requirements.
package calculator

import (
	"errors"
	"fmt"
	"log" // Keep log for warnings for now, can be removed later
	"math"
	"tfccalc/data" // Import the data definitions
)

// GetDefaultPercentages calculates the default percentages for an alloy's ingredients
// by taking the midpoint of their defined min/max range.
// It adjusts the first ingredient slightly if rounding causes the total not to be exactly 100%.
func GetDefaultPercentages(alloyID string) (map[string]float64, error) {
	alloy, ok := data.GetAlloyByID(alloyID)
	if !ok {
		return nil, fmt.Errorf("alloy %s not found", alloyID)
	}
	// Base materials or alloys without defined ingredients have no defaults.
	if len(alloy.Ingredients) == 0 {
		return make(map[string]float64), nil
	}

	percentages := make(map[string]float64)
	total := 0.0
	for _, ing := range alloy.Ingredients {
		mid := (ing.Min + ing.Max) / 2.0
		percentages[ing.ID] = mid
		total += mid
	}

	// Adjust if the total is not exactly 100 due to averaging
	if math.Abs(total-100.0) > 0.01 && len(alloy.Ingredients) > 0 {
		diff := 100.0 - total
		firstIngID := alloy.Ingredients[0].ID
		if _, exists := percentages[firstIngID]; exists {
			percentages[firstIngID] += diff
		} else {
			// This should not happen if Ingredients is not empty
			return nil, fmt.Errorf("internal error: first ingredient %s not found in percentage map during adjustment", firstIngID)
		}
	}
	return percentages, nil
}

// ValidatePercentages checks if a given map of percentages for an alloy is valid.
// It ensures all required ingredients are present, percentages are within the defined [min, max] range,
// and the total sum is exactly 100%.
// An empty or nil map is considered valid (implies default percentages should be used).
func ValidatePercentages(alloyID string, percentages map[string]float64) (bool, error) {
	alloy, ok := data.GetAlloyByID(alloyID)
	if !ok {
		return false, fmt.Errorf("alloy %s not found for validation", alloyID)
	}
	// No ingredients to validate, valid only if input map is also empty
	if len(alloy.Ingredients) == 0 {
		return len(percentages) == 0, nil
	}
	// If percentages map is empty/nil, it's valid (use defaults later)
	if len(percentages) == 0 {
		return true, nil
	}

	totalPercent := 0.0
	requiredIngCount := len(alloy.Ingredients)
	providedIngCount := len(percentages)

	// If a non-empty map is provided, it must contain exactly the required ingredients
	if providedIngCount != requiredIngCount {
		return false, fmt.Errorf("incorrect number of ingredients in percentage map (expected %d, got %d) for %s", requiredIngCount, providedIngCount, alloyID)
	}

	// Check each provided percentage
	for _, ingData := range alloy.Ingredients {
		percent, found := percentages[ingData.ID]
		if !found {
			// This check should be redundant if providedIngCount == requiredIngCount, but good practice
			return false, fmt.Errorf("internal error: ingredient %s percentage missing in map for %s", ingData.ID, alloyID)
		}

		// Allow for small floating point inaccuracies when comparing with min/max
		epsilon := 0.001
		if percent < ingData.Min-epsilon || percent > ingData.Max+epsilon {
			ingName := data.GetAlloyNameByID(ingData.ID) // Get display name for error
			return false, fmt.Errorf("percentage for %s (%.2f%%) is outside the allowed range [%.0f-%.0f%%] in alloy %s", ingName, percent, ingData.Min, ingData.Max, alloy.Name)
		}
		totalPercent += percent
	}

	// Check if the sum is very close to 100
	if math.Abs(totalPercent-100.0) > 0.01 {
		return false, fmt.Errorf("sum of percentages for %s (%.2f%%) does not equal 100%%", alloy.Name, totalPercent)
	}

	return true, nil // Percentages are valid
}

// sumMaterials merges two maps containing material amounts, summing the values for common keys.
func sumMaterials(materials1, materials2 map[string]float64) map[string]float64 {
	result := make(map[string]float64)
	// Copy first map
	for id, amount := range materials1 {
		result[id] = amount
	}
	// Add/Sum values from second map
	for id, amount := range materials2 {
		result[id] += amount // += 0 if key doesn't exist in result yet
	}
	return result
}

// getBaseMaterialBreakdown recursively calculates the required amounts of *base* materials
// (and pig iron) needed to produce a given amount of a target material/alloy.
// `allUserPercentages` contains user overrides for specific alloys in the hierarchy.
// `level` tracks recursion depth to prevent infinite loops.
func getBaseMaterialBreakdown(targetID string, amountMB float64, allUserPercentages map[string]map[string]float64, level int) (map[string]float64, error) {
	// Basic recursion depth check
	if level > 20 {
		return nil, errors.New("maximum recursion depth exceeded, check for cyclic dependencies")
	}

	targetData, ok := data.GetAlloyByID(targetID)
	if !ok {
		return nil, fmt.Errorf("unknown material/alloy ID: %s", targetID)
	}

	// --- Base Cases ---
	// 1. If it's a base metal, return the required amount of itself.
	if targetData.Type == "base" {
		return map[string]float64{targetID: amountMB}, nil
	}
	// 2. If it's steel, it requires pig iron. Recursively call for pig iron.
	if targetID == "steel" {
		// Pass percentages down, although pig_iron doesn't use them
		return getBaseMaterialBreakdown("pig_iron", amountMB, allUserPercentages, level+1)
	}

	// --- Recursive Cases ---

	// 3. Handle final steel when it appears as an *ingredient* in another recipe
	//    (e.g., Black Steel needed for Raw Blue Steel).
	//    Its cost is the sum of its raw form cost and its extra ingredient cost.
	if targetData.Type == "final_steel" {
		if targetData.RawForm == "" || targetData.ExtraIngredient == "" {
			return nil, fmt.Errorf("incomplete data for final steel %s (missing RawForm or ExtraIngredient)", targetID)
		}
		// Calculate cost of raw form part
		rawCost, err := getBaseMaterialBreakdown(targetData.RawForm, amountMB, allUserPercentages, level+1)
		if err != nil {
			return nil, fmt.Errorf("error calculating raw_form cost (%s) for %s: %w", targetData.RawForm, targetID, err)
		}
		// Calculate cost of extra ingredient part
		extraCost, err := getBaseMaterialBreakdown(targetData.ExtraIngredient, amountMB, allUserPercentages, level+1)
		if err != nil {
			return nil, fmt.Errorf("error calculating extra_ingredient cost (%s) for %s: %w", targetData.ExtraIngredient, targetID, err)
		}
		// Return the sum
		return sumMaterials(rawCost, extraCost), nil
	}

	// 4. Handle regular alloys, processed materials (like steel was handled above), and raw steels.
	if targetData.Type == "alloy" || targetData.Type == "processed" || targetData.Type == "raw_steel" {
		// If no ingredients defined (e.g., incomplete data), return empty map.
		if len(targetData.Ingredients) == 0 {
			return make(map[string]float64), nil
		}

		// Determine which percentages to use for *this specific alloy*
		percentagesToUse := make(map[string]float64)
		var err error
		useCustom := false

		// Check if user provided specific percentages for *this* alloy ID
		if specificUserPercentages, found := allUserPercentages[targetID]; found && len(specificUserPercentages) > 0 {
			// Attempt to fill in missing percentages with defaults before validation
			fullPercMap := make(map[string]float64)
			for k, v := range specificUserPercentages { fullPercMap[k] = v } // Copy user provided

			if len(fullPercMap) < len(targetData.Ingredients) {
				 defaultPercentages, defErr := GetDefaultPercentages(targetID)
				 // Only supplement if defaults were successfully retrieved
				 if defErr == nil && defaultPercentages != nil {
					 for _, ing := range targetData.Ingredients {
						  if _, exists := fullPercMap[ing.ID]; !exists {
							   fullPercMap[ing.ID] = defaultPercentages[ing.ID] // Add default for missing ingredient
						  }
					 }
				 }
			}

			// Validate the (potentially supplemented) percentage map
			valid, validationErr := ValidatePercentages(targetID, fullPercMap)
			if valid {
				percentagesToUse = fullPercMap // Use the valid user/supplemented map
				useCustom = true
			} else {
				// Log a warning if invalid user percentages were provided for this level
				log.Printf("Warning: Invalid user percentages provided for %s (%v), using defaults.", targetID, validationErr)
				// Fall through to use defaults (useCustom remains false)
			}
		}

		// If custom percentages were not used (either not provided or invalid), get defaults
		if !useCustom {
			percentagesToUse, err = GetDefaultPercentages(targetID)
			if err != nil {
				return nil, fmt.Errorf("error getting default percentages for %s: %w", targetData.Name, err)
			}
			// Validate the defaults just in case
			validDef, defValErr := ValidatePercentages(targetID, percentagesToUse)
			if !validDef {
				return nil, fmt.Errorf("default percentages for %s are invalid: %w", targetData.Name, defValErr)
			}
		}

		// Recursively calculate requirements for each ingredient based on determined percentages
		totalBaseMaterials := make(map[string]float64)
		for _, ingredient := range targetData.Ingredients {
			percentage, found := percentagesToUse[ingredient.ID]
			if !found {
				// This indicates an internal logic error if validation passed
				return nil, fmt.Errorf("internal error: percentage not found for %s in alloy %s after validation", ingredient.ID, targetID)
			}
			requiredIngAmountMB := amountMB * (percentage / 100.0)

			// Skip calculation if amount is negligible to avoid unnecessary recursion/float issues
			if requiredIngAmountMB < 0.001 {
				continue
			}

			// Recursive call - pass the *entire* allUserPercentages map down
			baseMaterialsForIngredient, err := getBaseMaterialBreakdown(ingredient.ID, requiredIngAmountMB, allUserPercentages, level+1)
			if err != nil {
				// Wrap error for better context
				return nil, fmt.Errorf("error calculating component %s for %s: %w", ingredient.ID, targetID, err)
			}
			// Add the results for this ingredient to the total
			totalBaseMaterials = sumMaterials(totalBaseMaterials, baseMaterialsForIngredient)
		}
		return totalBaseMaterials, nil
	}

	// Should not reach here if all material types are handled
	return nil, fmt.Errorf("unhandled material type: %s for %s", targetData.Type, targetID)
}

// CalculateRequirements is the main exported function to calculate alloy needs.
// It handles the top-level request, mode switching (mB vs. Ingots),
// and the special processing steps for final steels (adding extra ingredients).
// Returns: map of base material ID -> required mB, map of base material ID -> required ingots, error.
func CalculateRequirements(targetID string, amount float64, mode string, allUserPercentages map[string]map[string]float64) (map[string]float64, map[string]float64, error) {
	// --- Input Validation ---
	if amount <= 0 {
		return nil, nil, errors.New("amount must be positive")
	}
	if mode != "mB" && mode != "Ingots" {
		return nil, nil, errors.New("invalid calculation mode")
	}
	targetData, ok := data.GetAlloyByID(targetID)
	if !ok {
		return nil, nil, fmt.Errorf("alloy with ID '%s' not found", targetID)
	}

	finalMaterialsMB := make(map[string]float64)
	var err error

	// --- Validate Top-Level Percentages (if provided) ---
	// Determine which alloy's percentages need validation (target or its raw form)
	idForValidation := targetID
	if targetData.Type == "final_steel" {
		idForValidation = targetData.RawForm
	}
	// Check only if percentages for this specific alloy ID exist in the input map
	if specificPerc, found := allUserPercentages[idForValidation]; found && len(specificPerc) > 0 {
		// Attempt to fill in missing percentages with defaults before validation
		fullPercMap := make(map[string]float64)
		for k, v := range specificPerc { fullPercMap[k] = v }

		alloyToValidate, _ := data.GetAlloyByID(idForValidation)
		if len(fullPercMap) < len(alloyToValidate.Ingredients) {
			 defaultPercentages, _ := GetDefaultPercentages(idForValidation)
			 if defaultPercentages != nil {
				 for _, ing := range alloyToValidate.Ingredients {
					 if _, exists := fullPercMap[ing.ID]; !exists {
						 fullPercMap[ing.ID] = defaultPercentages[ing.ID]
					 }
				 }
			 }
		}
		// Validate the potentially completed map
		valid, validationErr := ValidatePercentages(idForValidation, fullPercMap)
		if !valid {
			// Return error if top-level percentages are invalid
			return nil, nil, fmt.Errorf("invalid user percentages for %s: %w", data.GetAlloyNameByID(idForValidation), validationErr)
		}
		// Update the map in allUserPercentages in case defaults were added (important for passing down)
        allUserPercentages[idForValidation] = fullPercMap
	} // End top-level validation

	// --- Calculation Logic based on Mode ---
	if mode == "Ingots" {
		amountMB := amount * 100.0

		// Special handling for final steels in Ingot mode
		if targetData.Type == "final_steel" {
			if targetData.RawForm == "" || targetData.ExtraIngredient == "" {
				return nil, nil, fmt.Errorf("incomplete data for final steel %s", targetID)
			}

			// 1. Calculate base materials for the RAW form part
			rawBreakdown, err := getBaseMaterialBreakdown(targetData.RawForm, amountMB, allUserPercentages, 0)
			if err != nil {
				return nil, nil, fmt.Errorf("error calculating raw form %s: %w", targetData.RawForm, err)
			}

			// 2. Calculate base materials for the EXTRA ingredient part (also scaled by amountMB)
			//    This recursive call handles cases like Blue Steel needing Black Steel.
			extraIngredientBreakdown := make(map[string]float64)
			if targetData.ExtraIngredient == "pig_iron" {
				// Pig iron is base, cost is just itself
				extraIngredientBreakdown = map[string]float64{"pig_iron": amountMB}
			} else {
				// For other extra ingredients (e.g., Black Steel for Blue Steel),
				// we need the cost equivalent to the *same amount* in mB.
				// We can use getBaseMaterialBreakdown directly here.
				extraIngredientCost, extraErr := getBaseMaterialBreakdown(targetData.ExtraIngredient, amountMB, allUserPercentages, 0)
				if extraErr != nil {
					 return nil, nil, fmt.Errorf("error calculating extra ingredient cost (%s): %w", targetData.ExtraIngredient, extraErr)
				}
				 extraIngredientBreakdown = extraIngredientCost
			}

			// 3. Sum the costs
			finalMaterialsMB = sumMaterials(rawBreakdown, extraIngredientBreakdown)

		} else {
			// For non-final-steels in Ingot mode, just calculate for amount * 100 mB
			finalMaterialsMB, err = getBaseMaterialBreakdown(targetID, amountMB, allUserPercentages, 0)
			if err != nil {
				return nil, nil, fmt.Errorf("error calculating %s: %w", targetData.Name, err)
			}
		}

	} else { // mode == "mB"
		// In mB mode, calculate the composition "as is".
		// For final steels, this means calculating the cost of their *raw form*.
		idToCalculate := targetID
		if targetData.Type == "final_steel" {
			idToCalculate = targetData.RawForm
		}
		finalMaterialsMB, err = getBaseMaterialBreakdown(idToCalculate, amount, allUserPercentages, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("error calculating %s: %w", data.GetAlloyNameByID(idToCalculate), err)
		}
	} // End calculation logic

	// --- Prepare Final Results ---
	finalMaterialsIngots := make(map[string]float64)
	resultMB := make(map[string]float64)

	// Filter out negligible amounts and calculate ingots
	for id, mb := range finalMaterialsMB {
		if mb > 0.001 { // Threshold for negligible amounts
			resultMB[id] = mb
			finalMaterialsIngots[id] = mb / 100.0
		}
	}

	// Handle edge case: calculation resulted in nothing (or only negligible amounts)
	if len(resultMB) == 0 {
		// If the target was a base metal in mB mode, the result is just itself
		if targetData.Type == "base" && mode == "mB" {
			resultMB[targetID] = amount
			finalMaterialsIngots[targetID] = amount / 100.0
			return resultMB, finalMaterialsIngots, nil
		}
		// Otherwise, maybe return an empty map or an error? Empty maps are probably fine.
	}

	return resultMB, finalMaterialsIngots, nil
}