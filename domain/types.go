package domain

// AlloyType defines known alloy/material categories.
type AlloyType string

const (
	AlloyTypeBase       AlloyType = "base"
	AlloyTypeAlloy      AlloyType = "alloy"
	AlloyTypeProcessed  AlloyType = "processed"
	AlloyTypeRawSteel   AlloyType = "raw_steel"
	AlloyTypeFinalSteel AlloyType = "final_steel"
)

// Ingredient describes a single ingredient with percentage bounds.
type Ingredient struct {
	IngredientID string
	Min          float64
	Max          float64
}

// Alloy represents a material or alloy definition.
type Alloy struct {
	ID                string
	Name              string
	Type              AlloyType
	RawFormID         *string
	ExtraIngredientID *string
	Ingredients       []Ingredient
}
