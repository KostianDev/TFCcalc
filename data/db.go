// tfccalc/data/db.go
package data

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/go-sql-driver/mysql"
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

// dbConn holds the global DB connection. Initialized by InitDB().
var (
	db             *sql.DB
	initOnce       sync.Once
	alloyCache     map[string]*AlloyInfo
	alloyCacheLock sync.RWMutex
)

// InitDB opens a connection to MySQL using the provided DSN.
// Call this once at program start (e.g. in main).
func InitDB(dsn string) error {
	var err error
	initOnce.Do(func() {
		db, err = sql.Open("mysql", dsn+"&parseTime=true&charset=utf8mb4")
		if err != nil {
			log.Printf("Error opening MySQL: %v", err)
			return
		}
		if pingErr := db.Ping(); pingErr != nil {
			err = fmt.Errorf("cannot ping MySQL: %w", pingErr)
			return
		}
		alloyCache = make(map[string]*AlloyInfo)
	})
	return err
}

// dbGetAlloyByID fetches a single AlloyInfo (including its ingredients) from DB by ID.
// Returns (AlloyInfo, true) if found, or (zero, false) otherwise.
func dbGetAlloyByID(id string) (AlloyInfo, bool) {
	// Check cache first
	alloyCacheLock.RLock()
	if info, ok := alloyCache[id]; ok {
		alloyCacheLock.RUnlock()
		return *info, true
	}
	alloyCacheLock.RUnlock()

	// Not in cache → fetch from DB
	queryAlloy := `
		SELECT id, name, type, raw_form_id, extra_ingredient_id
		FROM alloys
		WHERE id = ?
	`
	row := db.QueryRow(queryAlloy, id)
	var a AlloyInfo
	var rawForm sql.NullString
	var extraIng sql.NullString
	if err := row.Scan(&a.ID, &a.Name, &a.Type, &rawForm, &extraIng); err != nil {
		if err == sql.ErrNoRows {
			return AlloyInfo{}, false
		}
		log.Printf("Error querying alloy by ID %s: %v", id, err)
		return AlloyInfo{}, false
	}
	a.RawFormID = rawForm
	a.ExtraIngredientID = extraIng

	// Fetch ingredients
	a.Ingredients = dbGetIngredientsForAlloy(id)

	// Cache it
	alloyCacheLock.Lock()
	alloyCache[id] = &a
	alloyCacheLock.Unlock()
	return a, true
}

// dbGetAllAlloys returns a map[id] → AlloyInfo for all alloys in the database.
func dbGetAllAlloys() map[string]AlloyInfo {
	result := make(map[string]AlloyInfo)

	// If cache already populated for *all* IDs, return a copy
	alloyCacheLock.RLock()
	if len(alloyCache) > 0 {
		for k, v := range alloyCache {
			result[k] = *v
		}
		alloyCacheLock.RUnlock()
		return result
	}
	alloyCacheLock.RUnlock()

	// Otherwise, fetch all rows from `alloys`
	rows, err := db.Query(`SELECT id, name, type, raw_form_id, extra_ingredient_id FROM alloys`)
	if err != nil {
		log.Printf("Error querying all alloys: %v", err)
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var a AlloyInfo
		var rawForm sql.NullString
		var extraIng sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &rawForm, &extraIng); err != nil {
			log.Printf("Error scanning alloy row: %v", err)
			continue
		}
		a.RawFormID = rawForm
		a.ExtraIngredientID = extraIng
		a.Ingredients = dbGetIngredientsForAlloy(a.ID)

		// Populate cache + result
		alloyCacheLock.Lock()
		alloyCache[a.ID] = &a
		alloyCacheLock.Unlock()
		result[a.ID] = a
	}
	return result
}

// dbGetIngredientsForAlloy returns []IngredientInfo for a given alloy_id.
func dbGetIngredientsForAlloy(alloyID string) []IngredientInfo {
	query := `
		SELECT ingredient_id, min_pct, max_pct
		FROM ingredients
		WHERE alloy_id = ?
	`
	rows, err := db.Query(query, alloyID)
	if err != nil {
		log.Printf("Error querying ingredients for %s: %v", alloyID, err)
		return nil
	}
	defer rows.Close()

	var list []IngredientInfo
	for rows.Next() {
		var ing IngredientInfo
		if err := rows.Scan(&ing.IngredientID, &ing.Min, &ing.Max); err != nil {
			log.Printf("Error scanning ingredient row for %s: %v", alloyID, err)
			continue
		}
		list = append(list, ing)
	}
	return list
}
