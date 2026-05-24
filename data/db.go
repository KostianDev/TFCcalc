// tfccalc/data/db.go
package data

import (
	"database/sql"
	"sync"

	"tfccalc/domain"
	mysqlrepo "tfccalc/repository/mysql"
)

// AlloyInfo represents a single alloy/material row fetched from the database.
type AlloyInfo struct {
	ID                string
	Name              string
	Type              string // "base", "alloy", "processed", "raw_steel", "final_steel"
	RawFormID         sql.NullString
	ExtraIngredientID sql.NullString
	Ingredients       []IngredientInfo
}

// IngredientInfo represents one ingredient entry (alloy_id + ingredient_id + min/max).
type IngredientInfo struct {
	IngredientID string
	Min          float64
	Max          float64
}

// repo holds the global repository instance. Initialized by InitDB().
var (
	repo     domain.AlloyRepository
	initOnce sync.Once
)

// InitDB opens a connection to MySQL using the provided DSN.
// Call this once at program start (e.g. in main).
func InitDB(dsn string) error {
	var err error
	initOnce.Do(func() {
		repo, err = mysqlrepo.NewRepository(dsn)
	})
	return err
}
