// Package data defines the data structures and provides the dataset
// for TerraFirmaCraft alloys and their recipes.
package data

import "fmt"

// IngredientInfo describes a single ingredient within an alloy recipe,
// including its ID and allowed percentage range.
type IngredientInfo struct {
	ID  string  // Material/Alloy ID (key in TfcAlloys map)
	Min float64 // Minimum percentage allowed
	Max float64 // Maximum percentage allowed
}

// AlloyInfo holds all information about a specific alloy or base material.
type AlloyInfo struct {
	Name            string           // Displayable name (in English)
	Type            string           // Type identifier: "base", "alloy", "processed", "raw_steel", "final_steel"
	Ingredients     []IngredientInfo // List of ingredients required (for alloys/steels)
	RawForm         string           // ID of the raw form alloy (used for final_steel types)
	ExtraIngredient string           // ID of the extra ingredient needed for final processing (used for final_steel types)
}

// TfcAlloys is the main exported map containing all TFC alloy data used by the calculator.
// The map key is the internal ID string for the alloy or material.
var TfcAlloys = map[string]AlloyInfo{
	// Base Metals
	"copper":   {Name: "Copper", Type: "base"},
	"zinc":     {Name: "Zinc", Type: "base"},
	"bismuth":  {Name: "Bismuth", Type: "base"},
	"silver":   {Name: "Silver", Type: "base"},
	"gold":     {Name: "Gold", Type: "base"},
	"nickel":   {Name: "Nickel", Type: "base"},
	"pig_iron": {Name: "Pig Iron", Type: "base"},

	// Processed Material
	"steel": {
		Name:        "Steel",
		Type:        "processed",
		Ingredients: []IngredientInfo{{ID: "pig_iron", Min: 100, Max: 100}}, // Steel simply requires Pig Iron 1:1 for calculations
	},

	// Basic Alloys
	"bismuth_bronze": {
		Name: "Bismuth Bronze", Type: "alloy",
		Ingredients: []IngredientInfo{
			{ID: "zinc", Min: 20, Max: 30},
			{ID: "copper", Min: 50, Max: 65},
			{ID: "bismuth", Min: 10, Max: 20},
		},
	},
	"black_bronze": {
		Name: "Black Bronze", Type: "alloy",
		Ingredients: []IngredientInfo{
			{ID: "copper", Min: 50, Max: 70},
			{ID: "silver", Min: 10, Max: 25},
			{ID: "gold", Min: 10, Max: 25},
		},
	},
	"brass": {
		Name: "Brass", Type: "alloy",
		Ingredients: []IngredientInfo{
			{ID: "copper", Min: 88, Max: 92},
			{ID: "zinc", Min: 8, Max: 12},
		},
	},
	"rose_gold": {
		Name: "Rose Gold", Type: "alloy",
		Ingredients: []IngredientInfo{
			{ID: "copper", Min: 15, Max: 30},
			{ID: "gold", Min: 70, Max: 85},
		},
	},
	"sterling_silver": {
		Name: "Sterling Silver", Type: "alloy",
		Ingredients: []IngredientInfo{
			{ID: "copper", Min: 20, Max: 40},
			{ID: "silver", Min: 60, Max: 80},
		},
	},

	// --- Steels ---
	// Raw Steels (Intermediates)
	"raw_black_steel": {
		Name: "Raw Black Steel", Type: "raw_steel",
		Ingredients: []IngredientInfo{
			{ID: "steel", Min: 50, Max: 70},
			{ID: "nickel", Min: 15, Max: 25},
			{ID: "black_bronze", Min: 15, Max: 25},
		},
	},
	"raw_blue_steel": {
		Name: "Raw Blue Steel", Type: "raw_steel",
		Ingredients: []IngredientInfo{
			// Note: Requires final Black Steel as an ingredient. Calculation logic handles this.
			{ID: "black_steel", Min: 50, Max: 55},
			{ID: "steel", Min: 20, Max: 25},
			{ID: "bismuth_bronze", Min: 10, Max: 15},
			{ID: "sterling_silver", Min: 10, Max: 15},
		},
	},
	"raw_red_steel": {
		Name: "Raw Red Steel", Type: "raw_steel",
		Ingredients: []IngredientInfo{
			// Note: Requires final Black Steel as an ingredient. Calculation logic handles this.
			{ID: "black_steel", Min: 50, Max: 55},
			{ID: "steel", Min: 20, Max: 25},
			{ID: "brass", Min: 10, Max: 15},
			{ID: "rose_gold", Min: 10, Max: 15},
		},
	},

	// Final Steels (Selectable Targets)
	"black_steel": {
		Name:            "Black Steel",
		Type:            "final_steel",
		RawForm:         "raw_black_steel", // The raw form needed
		ExtraIngredient: "pig_iron",        // The extra item added (1:1 by ingot)
	},
	"blue_steel": {
		Name:            "Blue Steel",
		Type:            "final_steel",
		RawForm:         "raw_blue_steel",
		ExtraIngredient: "black_steel", // Requires final Black Steel
	},
	"red_steel": {
		Name:            "Red Steel",
		Type:            "final_steel",
		RawForm:         "raw_red_steel",
		ExtraIngredient: "black_steel", // Requires final Black Steel
	},
}

// GetAlloyByID provides safe access to the alloy data map.
// It returns the AlloyInfo and true if found, otherwise a zero value and false.
func GetAlloyByID(id string) (AlloyInfo, bool) {
	alloy, ok := TfcAlloys[id]
	return alloy, ok
}

// GetAlloyNameByID returns the display name for a given material ID.
// Returns a fallback string if the ID is not found.
func GetAlloyNameByID(id string) string {
	if alloy, ok := TfcAlloys[id]; ok {
		return alloy.Name
	}
	// Fallback if ID is somehow unknown
	return fmt.Sprintf("Unknown[%s]", id)
}