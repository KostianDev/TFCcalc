// data/data_test.go
package data

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		"tfccalc_user", "tfccalc_pass", "127.0.0.1", 3306, "tfccalc_db",
	)
	if err := InitDB(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize DB: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestGetAlloyByID_ExistsAndNotExists(t *testing.T) {
	// Check an existing alloy
	alloy, ok := GetAlloyByID("brass")
	if !ok {
		t.Fatalf("GetAlloyByID(brass) returned ok=false, want true")
	}
	if alloy.Name != "Brass" || alloy.Type != "alloy" {
		t.Errorf("GetAlloyByID(brass) = %+v, want Name=\"Brass\", Type=\"alloy\"", alloy)
	}
	// Check a non-existent ID
	_, ok2 := GetAlloyByID("nonexistent_id")
	if ok2 {
		t.Errorf("GetAlloyByID(nonexistent_id) = ok=true, want ok=false")
	}
}

func TestGetAllAlloys_BasicConsistency(t *testing.T) {
	allAlloys := GetAllAlloys()
	// Ensure there is at least one alloy in the DB
	if len(allAlloys) == 0 {
		t.Fatalf("GetAllAlloys returned 0 entries, want > 0")
	}
	// For each ID returned, GetAlloyByID should find it.
	for id := range allAlloys {
		if _, ok := GetAlloyByID(id); !ok {
			t.Errorf("GetAllAlloys returned ID %q that GetAlloyByID cannot find", id)
		}
	}
}

func TestGetAlloyByID_Caching(t *testing.T) {
	// Two calls to GetAlloyByID should return identical data without error
	a1, ok1 := GetAlloyByID("brass")
	a2, ok2 := GetAlloyByID("brass")
	if !ok1 || !ok2 {
		t.Fatalf("GetAlloyByID(brass) returned ok=false")
	}
	if a1.ID != a2.ID || a1.Name != a2.Name || a1.Type != a2.Type {
		t.Errorf("Cached GetAlloyByID returned different results: %+v vs %+v", a1, a2)
	}
}

func TestGetAlloyNameByID(t *testing.T) {
	name := GetAlloyNameByID("brass")
	if name != "Brass" {
		t.Errorf("GetAlloyNameByID(brass) = %q, want \"Brass\"", name)
	}
	unknown := GetAlloyNameByID("does_not_exist")
	if len(unknown) == 0 || unknown[:7] != "Unknown" {
		t.Errorf("GetAlloyNameByID(does_not_exist) = %q, want prefix \"Unknown\"", unknown)
	}
}
