package calculator

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"tfccalc/data"
	"time"
)

// TestMain sets up the shared DB connection for all tests.
func TestMain(m *testing.M) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		"tfccalc_user", "tfccalc_pass", "127.0.0.1", 3306, "tfccalc_db",
	)
	if err := data.InitDB(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize DB: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// floatMapEqual compares two maps[string]float64 within a tolerance.
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

func TestGetDefaultPercentages_Brass(t *testing.T) {
	// For "brass": copper [88,92] and zinc [8,12].
	// Midpoints: copper=90, zinc=10
	want := map[string]float64{
		"copper": 90.0,
		"zinc":   10.0,
	}
	got, err := GetDefaultPercentages("brass")
	if err != nil {
		t.Fatalf("GetDefaultPercentages(brass) returned error: %v", err)
	}
	if !floatMapEqual(got, want, 0.0001) {
		t.Errorf("GetDefaultPercentages(brass) = %v, want %v", got, want)
	}
}

func TestValidatePercentages_ValidAndInvalid(t *testing.T) {
	// Valid percentages: copper=90, zinc=10
	valid := map[string]float64{"copper": 90.0, "zinc": 10.0}
	ok, err := ValidatePercentages("brass", valid)
	if !ok || err != nil {
		t.Errorf("ValidatePercentages(valid) = (%v,%v), want (true,nil)", ok, err)
	}

	// Missing key: only copper
	missing := map[string]float64{"copper": 90.0}
	ok2, _ := ValidatePercentages("brass", missing)
	if ok2 {
		t.Errorf("ValidatePercentages(missing) = true, want false")
	}

	// Out of range: copper=95, zinc=5
	outOfRange := map[string]float64{"copper": 95.0, "zinc": 5.0}
	ok3, _ := ValidatePercentages("brass", outOfRange)
	if ok3 {
		t.Errorf("ValidatePercentages(outOfRange) = true, want false")
	}

	// Sum not equal to 100: copper=80, zinc=10
	sumWrong := map[string]float64{"copper": 80.0, "zinc": 10.0}
	ok4, _ := ValidatePercentages("brass", sumWrong)
	if ok4 {
		t.Errorf("ValidatePercentages(sumWrong) = true, want false")
	}
}

func TestResolvePercentagesForAlloy_CustomAndDefaults(t *testing.T) {
	// Case A: empty userPerc → defaults
	gotA, errA := ResolvePercentagesForAlloy("brass", nil)
	if errA != nil {
		t.Fatalf("ResolvePercentagesForAlloy(empty) returned error: %v", errA)
	}
	wantDefault := map[string]float64{"copper": 90.0, "zinc": 10.0}
	if !floatMapEqual(gotA, wantDefault, 0.0001) {
		t.Errorf("ResolvePercentagesForAlloy(empty) = %v, want %v", gotA, wantDefault)
	}

	// Case B: partial user map → sum 102 → invalid → defaults
	userB := map[string]float64{"copper": 92.0}
	gotB, errB := ResolvePercentagesForAlloy("brass", userB)
	if errB != nil {
		t.Fatalf("ResolvePercentagesForAlloy(partial) returned error: %v", errB)
	}
	if !floatMapEqual(gotB, wantDefault, 0.0001) {
		t.Errorf("ResolvePercentagesForAlloy(partial) = %v, want %v", gotB, wantDefault)
	}

	// Case C: out of range → invalid → defaults
	userC := map[string]float64{"copper": 200.0, "zinc": 0.0}
	gotC, errC := ResolvePercentagesForAlloy("brass", userC)
	if errC != nil {
		t.Fatalf("ResolvePercentagesForAlloy(invalid) returned error: %v", errC)
	}
	if !floatMapEqual(gotC, wantDefault, 0.0001) {
		t.Errorf("ResolvePercentagesForAlloy(invalid) = %v, want %v", gotC, wantDefault)
	}
}

func TestSumMaterials(t *testing.T) {
	m1 := map[string]float64{"a": 10.0, "b": 5.0}
	m2 := map[string]float64{"b": 2.5, "c": 7.5}
	got := sumMaterials(m1, m2)
	want := map[string]float64{"a": 10.0, "b": 7.5, "c": 7.5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sumMaterials(%v,%v) = %v, want %v", m1, m2, got, want)
	}
}

func TestGetBaseMaterialBreakdown_SimpleAndNested(t *testing.T) {
	// Base: "copper" → itself
	baseRes, errBase := getBaseMaterialBreakdown("copper", 50.0, nil, 0)
	if errBase != nil {
		t.Fatalf("getBaseMaterialBreakdown(base) error: %v", errBase)
	}
	wantBase := map[string]float64{"copper": 50.0}
	if !floatMapEqual(baseRes, wantBase, 0.0001) {
		t.Errorf("getBaseMaterialBreakdown(copper) = %v, want %v", baseRes, wantBase)
	}

	// Alloy: "brass" 100mB → 90 copper, 10 zinc
	alloyRes, errAlloy := getBaseMaterialBreakdown("brass", 100.0, nil, 0)
	if errAlloy != nil {
		t.Fatalf("getBaseMaterialBreakdown(brass) error: %v", errAlloy)
	}
	wantAlloy := map[string]float64{"copper": 90.0, "zinc": 10.0}
	if !floatMapEqual(alloyRes, wantAlloy, 0.0001) {
		t.Errorf("getBaseMaterialBreakdown(brass) = %v, want %v", alloyRes, wantAlloy)
	}

	// Nested: "black_steel" 100mB
	// raw_black_steel breakdown: steel=60→pig_iron=60, nickel=20, black_bronze=20→copper=12,zinc=4,nickel=4
	// totals: pig_iron=60, nickel=24, copper=12, zinc=4; extra pig_iron=100 → pig_iron=160
	res, errNested := getBaseMaterialBreakdown("black_steel", 100.0, nil, 0)
	if errNested != nil {
		t.Fatalf("getBaseMaterialBreakdown(black_steel) error: %v", errNested)
	}
	wantNested := map[string]float64{
		"pig_iron": 160.0,
		"nickel":   24.0,
		"copper":   12.0,
		"zinc":     4.0,
	}
	if !floatMapEqual(res, wantNested, 0.0001) {
		t.Errorf("getBaseMaterialBreakdown(black_steel) = %v, want %v", res, wantNested)
	}
}

func TestCalculateRequirements_Brass_And_BlackSteel(t *testing.T) {
	// Brass, 100 Ingots → 100*100mB=10000mB → 9000 copper, 1000 zinc
	mbMap, ingMap, err := CalculateRequirements("brass", 100.0, "Ingots", nil)
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

	// Black steel, 50mB
	// raw_black_steel(50): steel=30→pig_iron=30, nickel=10, black_bronze=10→copper=6,zinc=2,nickel=2
	// totals: pig_iron=30, nickel=12, copper=6, zinc=2; extra pig_iron=50→pig_iron=80
	mbMap2, ingMap2, err2 := CalculateRequirements("black_steel", 50.0, "mB", nil)
	if err2 != nil {
		t.Fatalf("CalculateRequirements(black_steel) error: %v", err2)
	}
	wantMB2 := map[string]float64{
		"pig_iron": 80.0,
		"nickel":   12.0,
		"copper":   6.0,
		"zinc":     2.0,
	}
	wantIng2 := map[string]float64{
		"pig_iron": 0.80,
		"nickel":   0.12,
		"copper":   0.06,
		"zinc":     0.02,
	}
	if !floatMapEqual(mbMap2, wantMB2, 0.001) {
		t.Errorf("CalculateRequirements(black_steel).MB = %v, want %v", mbMap2, wantMB2)
	}
	if !floatMapEqual(ingMap2, wantIng2, 0.0001) {
		t.Errorf("CalculateRequirements(black_steel).Ing = %v, want %v", ingMap2, wantIng2)
	}
}

// Test for invalid inputs to CalculateRequirements.
func TestCalculateRequirements_ErrorCases(t *testing.T) {
	// Amount ≤ 0 should return an error.
	_, _, err1 := CalculateRequirements("brass", 0, "mB", nil)
	if err1 == nil || err1.Error() != "amount must be positive" {
		t.Errorf("CalculateRequirements(brass, 0, …) error = %v, want \"amount must be positive\"", err1)
	}
	_, _, err2 := CalculateRequirements("brass", -5, "mB", nil)
	if err2 == nil || err2.Error() != "amount must be positive" {
		t.Errorf("CalculateRequirements(brass, -5, …) error = %v, want \"amount must be positive\"", err2)
	}

	// Invalid mode should return an error.
	_, _, err3 := CalculateRequirements("brass", 10, "WrongMode", nil)
	expectedModeErr := `invalid mode; only "mB" or "Ingots"`
	if err3 == nil || err3.Error() != expectedModeErr {
		t.Errorf("CalculateRequirements(brass, 10, WrongMode) error = %v, want %q", err3, expectedModeErr)
	}

	// Nonexistent alloy ID should return an error.
	_, _, err4 := CalculateRequirements("nonexistent", 10, "mB", nil)
	expectedAlloyErr := "alloy nonexistent not found"
	if err4 == nil || err4.Error() != expectedAlloyErr {
		t.Errorf("CalculateRequirements(nonexistent, 10, mB) error = %v, want %q", err4, expectedAlloyErr)
	}
}

// Test boundary conditions for ValidatePercentages.
func TestValidatePercentages_Boundaries(t *testing.T) {
	// Exact minimum values.
	validMin := map[string]float64{"copper": 88.0, "zinc": 12.0}
	ok1, err1 := ValidatePercentages("brass", validMin)
	if !ok1 || err1 != nil {
		t.Errorf("ValidatePercentages(boundary min) = (%v,%v), want (true,nil)", ok1, err1)
	}

	// Exact maximum values.
	validMax := map[string]float64{"copper": 92.0, "zinc": 8.0}
	ok2, err2 := ValidatePercentages("brass", validMax)
	if !ok2 || err2 != nil {
		t.Errorf("ValidatePercentages(boundary max) = (%v,%v), want (true,nil)", ok2, err2)
	}

	// Sum within EPS: 89.999 + 10.001 = 100.000
	almost := map[string]float64{"copper": 89.999, "zinc": 10.001}
	ok3, err3 := ValidatePercentages("brass", almost)
	if !ok3 || err3 != nil {
		t.Errorf("ValidatePercentages(almost sum 100) = (%v,%v), want (true,nil)", ok3, err3)
	}
}

// Test that an exact user map is returned unchanged.
func TestResolvePercentagesForAlloy_ExactUserMap(t *testing.T) {
	user := map[string]float64{"copper": 90.0, "zinc": 10.0}
	got, err := ResolvePercentagesForAlloy("brass", user)
	if err != nil {
		t.Fatalf("ResolvePercentagesForAlloy(exact) returned error: %v", err)
	}
	if !floatMapEqual(got, user, 0.0001) {
		t.Errorf("ResolvePercentagesForAlloy(exact) = %v, want %v", got, user)
	}
}

// Test that an empty (non-nil) user map falls back to defaults.
func TestResolvePercentagesForAlloy_EmptyMap(t *testing.T) {
	got, err := ResolvePercentagesForAlloy("brass", map[string]float64{})
	if err != nil {
		t.Fatalf("ResolvePercentagesForAlloy(empty map) returned error: %v", err)
	}
	want := map[string]float64{"copper": 90.0, "zinc": 10.0}
	if !floatMapEqual(got, want, 0.0001) {
		t.Errorf("ResolvePercentagesForAlloy(empty map) = %v, want %v", got, want)
	}
}

// Test that “steel” is handled inside getBaseMaterialBreakdown.
func TestGetBaseMaterialBreakdown_SteelInsideAlloy(t *testing.T) {
	// raw_black_steel(100): steel=60→pig_iron=60, nickel=20, black_bronze=20→copper=12,zinc=4,nickel=4
	res, err := getBaseMaterialBreakdown("raw_black_steel", 100.0, nil, 0)
	if err != nil {
		t.Fatalf("getBaseMaterialBreakdown(raw_black_steel) returned error: %v", err)
	}
	want := map[string]float64{"pig_iron": 60.0, "nickel": 24.0, "copper": 12.0, "zinc": 4.0}
	if !floatMapEqual(res, want, 0.0001) {
		t.Errorf("getBaseMaterialBreakdown(raw_black_steel) = %v, want %v", res, want)
	}
}

// TestRandomValidatePercentages picks random percentage maps for "brass" and checks ValidatePercentages.
// It ensures that any map drawn uniformly between 0–100 for each ingredient either
// (a) passes exactly when it lies within [Min,Max] and sums ≈100, or
// (b) fails otherwise.
func TestRandomValidatePercentages(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	const iterations = 500
	// Known ranges for "brass": copper ∈ [88,92], zinc ∈ [8,12]
	for i := 0; i < iterations; i++ {
		cu := rand.Float64() * 100.0 // 0..100
		zn := 100.0 - cu             // so they always sum exactly 100
		m := map[string]float64{"copper": cu, "zinc": zn}
		ok, _ := ValidatePercentages("brass", m)

		// The only way it should pass is if cu∈[88,92] and zn∈[8,12] (and they sum=100).
		inside := (cu >= 88.0 && cu <= 92.0) && (zn >= 8.0 && zn <= 12.0)
		if ok != inside {
			t.Errorf("iter %d: ValidatePercentages(brass, %#v) = %v, want %v", i, m, ok, inside)
		}
	}
}

// TestRandomCalculateBreakdown picks a random positive amount (0 < amt ≤ 1000),
// calls getBaseMaterialBreakdown("brass", amt, nil, 0), and then checks that
// the returned base‐metal totals sum exactly to amt and that no negative values appear.
func TestRandomCalculateBreakdown(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	const iterations = 200
	for i := 0; i < iterations; i++ {
		amt := rand.Float64()*999.0 + 1.0 // 1…1000 mB
		m, err := getBaseMaterialBreakdown("brass", amt, nil, 0)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		var sum float64
		for k, v := range m {
			if v < 0 {
				t.Errorf("iteration %d: negative amount %f for %q", i, v, k)
			}
			sum += v
		}
		// Because brass always splits exactly 90%/10%, sum should equal amt (within tiny epsilon).
		if diff := sum - amt; diff < -1e-6 || diff > 1e-6 {
			t.Errorf("iteration %d: sum of breakdown = %f, want %f", i, sum, amt)
		}
	}
}
