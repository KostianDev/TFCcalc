package alloy

import (
	"reflect"
	"testing"
	"tfccalc/domain"
)

type fakeRepo struct {
	alloys map[string]domain.Alloy
}

func (f *fakeRepo) GetAlloyByID(id string) (domain.Alloy, bool) {
	alloy, ok := f.alloys[id]
	return alloy, ok
}

func (f *fakeRepo) GetAllAlloys() map[string]domain.Alloy {
	out := make(map[string]domain.Alloy, len(f.alloys))
	for id, alloy := range f.alloys {
		out[id] = alloy
	}
	return out
}

func (f *fakeRepo) GetAlloyNameByID(id string) string {
	if alloy, ok := f.alloys[id]; ok {
		return alloy.Name
	}
	return ""
}

func buildTestRepo() *fakeRepo {
	alloys := map[string]domain.Alloy{
		"copper": {ID: "copper", Name: "Copper", Type: domain.AlloyTypeBase},
		"zinc":   {ID: "zinc", Name: "Zinc", Type: domain.AlloyTypeBase},
		"bismuth": {
			ID:   "bismuth",
			Name: "Bismuth",
			Type: domain.AlloyTypeBase,
		},
		"silver": {ID: "silver", Name: "Silver", Type: domain.AlloyTypeBase},
		"gold":   {ID: "gold", Name: "Gold", Type: domain.AlloyTypeBase},
		"nickel": {ID: "nickel", Name: "Nickel", Type: domain.AlloyTypeBase},
		"pig_iron": {
			ID:   "pig_iron",
			Name: "Pig Iron",
			Type: domain.AlloyTypeBase,
		},
		"black_bronze": {
			ID:   "black_bronze",
			Name: "Black Bronze",
			Type: domain.AlloyTypeAlloy,
			Ingredients: []domain.Ingredient{
				{IngredientID: "copper", Min: 50, Max: 70},
				{IngredientID: "silver", Min: 10, Max: 25},
				{IngredientID: "gold", Min: 10, Max: 25},
			},
		},
		"brass": {
			ID:   "brass",
			Name: "Brass",
			Type: domain.AlloyTypeAlloy,
			Ingredients: []domain.Ingredient{
				{IngredientID: "copper", Min: 88, Max: 92},
				{IngredientID: "zinc", Min: 8, Max: 12},
			},
		},
		"steel": {
			ID:   "steel",
			Name: "Steel",
			Type: domain.AlloyTypeProcessed,
			Ingredients: []domain.Ingredient{
				{IngredientID: "pig_iron", Min: 100, Max: 100},
			},
		},
		"raw_black_steel": {
			ID:   "raw_black_steel",
			Name: "Raw Black Steel",
			Type: domain.AlloyTypeRawSteel,
			Ingredients: []domain.Ingredient{
				{IngredientID: "steel", Min: 50, Max: 70},
				{IngredientID: "nickel", Min: 15, Max: 25},
				{IngredientID: "black_bronze", Min: 15, Max: 25},
			},
		},
		"black_steel": {
			ID:          "black_steel",
			Name:        "Black Steel",
			Type:        domain.AlloyTypeFinalSteel,
			Ingredients: []domain.Ingredient{},
		},
	}
	raw := "raw_black_steel"
	extra := "pig_iron"
	blackSteel := alloys["black_steel"]
	blackSteel.RawFormID = &raw
	blackSteel.ExtraIngredientID = &extra
	alloys["black_steel"] = blackSteel

	return &fakeRepo{alloys: alloys}
}

func TestGetDefaultPercentages_Brass(t *testing.T) {
	svc := NewService(buildTestRepo())
	got, err := svc.GetDefaultPercentages("brass")
	if err != nil {
		t.Fatalf("GetDefaultPercentages(brass) error: %v", err)
	}
	want := map[string]float64{"copper": 90.0, "zinc": 10.0}
	if !floatMapEqual(got, want, 0.0001) {
		t.Errorf("GetDefaultPercentages(brass) = %v, want %v", got, want)
	}
}

func TestValidatePercentages_Brass(t *testing.T) {
	svc := NewService(buildTestRepo())
	valid := map[string]float64{"copper": 90.0, "zinc": 10.0}
	ok, err := svc.ValidatePercentages("brass", valid)
	if !ok || err != nil {
		t.Errorf("ValidatePercentages(valid) = (%v,%v), want (true,nil)", ok, err)
	}

	missing := map[string]float64{"copper": 90.0}
	ok2, _ := svc.ValidatePercentages("brass", missing)
	if ok2 {
		t.Errorf("ValidatePercentages(missing) = true, want false")
	}
}

func TestCalculateRequirements_BrassAndBlackSteel(t *testing.T) {
	svc := NewService(buildTestRepo())
	mbMap, ingMap, err := svc.CalculateRequirements("brass", 100.0, "Ingots", nil)
	if err != nil {
		t.Fatalf("CalculateRequirements(brass) error: %v", err)
	}
	wantMB := map[string]float64{"copper": 9000.0, "zinc": 1000.0}
	wantIng := map[string]float64{"copper": 90.0, "zinc": 10.0}
	if !floatMapEqual(mbMap, wantMB, 0.001) {
		t.Errorf("CalculateRequirements(brass).MB = %v, want %v", mbMap, wantMB)
	}
	if !floatMapEqual(ingMap, wantIng, 0.001) {
		t.Errorf("CalculateRequirements(brass).Ing = %v, want %v", ingMap, wantIng)
	}

	mbMap2, ingMap2, err2 := svc.CalculateRequirements("black_steel", 50.0, "mB", nil)
	if err2 != nil {
		t.Fatalf("CalculateRequirements(black_steel) error: %v", err2)
	}
	wantMB2 := map[string]float64{
		"pig_iron": 80.0,
		"nickel":   10.0,
		"copper":   6.5,
		"silver":   1.75,
		"gold":     1.75,
	}
	wantIng2 := map[string]float64{
		"pig_iron": 0.80,
		"nickel":   0.10,
		"copper":   0.065,
		"silver":   0.0175,
		"gold":     0.0175,
	}
	if !floatMapEqual(mbMap2, wantMB2, 0.001) {
		t.Errorf("CalculateRequirements(black_steel).MB = %v, want %v", mbMap2, wantMB2)
	}
	if !floatMapEqual(ingMap2, wantIng2, 0.0001) {
		t.Errorf("CalculateRequirements(black_steel).Ing = %v, want %v", ingMap2, wantIng2)
	}
}

func TestResolvePercentagesForAlloy_ExactUserMap(t *testing.T) {
	svc := NewService(buildTestRepo())
	user := map[string]float64{"copper": 90.0, "zinc": 10.0}
	got, err := svc.ResolvePercentagesForAlloy("brass", user)
	if err != nil {
		t.Fatalf("ResolvePercentagesForAlloy(exact) error: %v", err)
	}
	if !floatMapEqual(got, user, 0.0001) {
		t.Errorf("ResolvePercentagesForAlloy(exact) = %v, want %v", got, user)
	}
}

func TestCalculateRequirements_ErrorCases(t *testing.T) {
	svc := NewService(buildTestRepo())
	_, _, err1 := svc.CalculateRequirements("brass", 0, "mB", nil)
	if err1 == nil || err1.Error() != "amount must be positive" {
		t.Errorf("CalculateRequirements(brass, 0) error = %v, want amount must be positive", err1)
	}
	_, _, err2 := svc.CalculateRequirements("brass", 10, "WrongMode", nil)
	if err2 == nil {
		t.Errorf("CalculateRequirements(brass, WrongMode) error = %v, want error", err2)
	}
	_, _, err3 := svc.CalculateRequirements("nonexistent", 10, "mB", nil)
	if err3 == nil {
		t.Errorf("CalculateRequirements(nonexistent) error = %v, want error", err3)
	}
}

func floatMapEqual(a, b map[string]float64, eps float64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if (va-vb) > eps || (vb-va) > eps {
			return false
		}
	}
	return true
}

func TestFakeRepoImplementsInterface(t *testing.T) {
	var _ domain.AlloyRepository = (*fakeRepo)(nil)
}

func TestGetAllAlloys_ReturnsCopy(t *testing.T) {
	repo := buildTestRepo()
	svc := NewService(repo)
	all := svc.GetAllAlloys()
	if !reflect.DeepEqual(all, repo.alloys) {
		t.Errorf("GetAllAlloys() mismatch")
	}
}
