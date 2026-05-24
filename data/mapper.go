package data

import (
	"database/sql"
	"tfccalc/domain"
)

func toAlloyInfo(a domain.Alloy) AlloyInfo {
	var rawForm sql.NullString
	if a.RawFormID != nil {
		rawForm = sql.NullString{String: *a.RawFormID, Valid: true}
	}

	var extraIng sql.NullString
	if a.ExtraIngredientID != nil {
		extraIng = sql.NullString{String: *a.ExtraIngredientID, Valid: true}
	}

	ingredients := make([]IngredientInfo, 0, len(a.Ingredients))
	for _, ing := range a.Ingredients {
		ingredients = append(ingredients, IngredientInfo{
			IngredientID: ing.IngredientID,
			Min:          ing.Min,
			Max:          ing.Max,
		})
	}

	return AlloyInfo{
		ID:                a.ID,
		Name:              a.Name,
		Type:              string(a.Type),
		RawFormID:         rawForm,
		ExtraIngredientID: extraIng,
		Ingredients:       ingredients,
	}
}

func toAlloyInfoMap(in map[string]domain.Alloy) map[string]AlloyInfo {
	out := make(map[string]AlloyInfo, len(in))
	for id, alloy := range in {
		out[id] = toAlloyInfo(alloy)
	}
	return out
}
