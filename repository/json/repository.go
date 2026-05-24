package jsonrepo

import (
	"encoding/json"
	"fmt"
	"os"
	"tfccalc/domain"
)

type jsonAlloy struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Type              string           `json:"type"`
	RawFormID         *string          `json:"raw_form_id"`
	ExtraIngredientID *string          `json:"extra_ingredient_id"`
	Ingredients       []jsonIngredient `json:"ingredients"`
}

type jsonIngredient struct {
	IngredientID string  `json:"ingredient_id"`
	Min          float64 `json:"min_pct"`
	Max          float64 `json:"max_pct"`
}

type jsonPayload struct {
	Alloys []jsonAlloy `json:"alloys"`
}

// Repository implements AlloyRepository backed by a JSON file.
type Repository struct {
	allByID map[string]domain.Alloy
}

// NewRepositoryFromFile loads alloy data from a JSON file.
func NewRepositoryFromFile(path string) (*Repository, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json file: %w", err)
	}

	var payload jsonPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("parse json file: %w", err)
	}

	allByID := make(map[string]domain.Alloy, len(payload.Alloys))
	for _, row := range payload.Alloys {
		if row.ID == "" {
			return nil, fmt.Errorf("json alloy has empty id")
		}
		if row.Name == "" {
			return nil, fmt.Errorf("json alloy %s has empty name", row.ID)
		}
		if _, exists := allByID[row.ID]; exists {
			return nil, fmt.Errorf("duplicate alloy id %s in json", row.ID)
		}

		ingredients := make([]domain.Ingredient, 0, len(row.Ingredients))
		for _, ing := range row.Ingredients {
			if ing.IngredientID == "" {
				return nil, fmt.Errorf("json alloy %s has ingredient with empty id", row.ID)
			}
			ingredients = append(ingredients, domain.Ingredient{
				IngredientID: ing.IngredientID,
				Min:          ing.Min,
				Max:          ing.Max,
			})
		}

		allByID[row.ID] = domain.Alloy{
			ID:                row.ID,
			Name:              row.Name,
			Type:              domain.AlloyType(row.Type),
			RawFormID:         row.RawFormID,
			ExtraIngredientID: row.ExtraIngredientID,
			Ingredients:       ingredients,
		}
	}

	return &Repository{allByID: allByID}, nil
}

// GetAlloyByID returns a single alloy by ID.
func (r *Repository) GetAlloyByID(id string) (domain.Alloy, bool) {
	alloy, ok := r.allByID[id]
	return alloy, ok
}

// GetAllAlloys returns all alloys.
func (r *Repository) GetAllAlloys() map[string]domain.Alloy {
	out := make(map[string]domain.Alloy, len(r.allByID))
	for id, alloy := range r.allByID {
		out[id] = alloy
	}
	return out
}

// GetAlloyNameByID returns the name for a given ID.
func (r *Repository) GetAlloyNameByID(id string) string {
	if alloy, ok := r.allByID[id]; ok {
		return alloy.Name
	}
	return ""
}
