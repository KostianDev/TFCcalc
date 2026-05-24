package mysqlrepo

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"tfccalc/domain"

	_ "github.com/go-sql-driver/mysql"
)

// Repository implements AlloyRepository using a MySQL backend.
type Repository struct {
	db        *sql.DB
	cache     map[string]domain.Alloy
	cacheLock sync.RWMutex
}

// NewRepository opens a MySQL connection and returns a repository instance.
func NewRepository(dsn string) (*Repository, error) {
	db, err := sql.Open(
		"mysql",
		dsn+"?parseTime=true&charset=utf8mb4&allowNativePasswords=true",
	)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if pingErr := db.Ping(); pingErr != nil {
		return nil, fmt.Errorf("cannot ping MySQL: %w", pingErr)
	}

	return &Repository{
		db:    db,
		cache: make(map[string]domain.Alloy),
	}, nil
}

// GetAlloyByID returns a single alloy by ID, including its ingredients.
func (r *Repository) GetAlloyByID(id string) (domain.Alloy, bool) {
	r.cacheLock.RLock()
	if info, ok := r.cache[id]; ok {
		r.cacheLock.RUnlock()
		return info, true
	}
	r.cacheLock.RUnlock()

	queryAlloy := `
		SELECT id, name, type, raw_form_id, extra_ingredient_id
		FROM alloys
		WHERE id = ?
	`
	row := r.db.QueryRow(queryAlloy, id)

	var a domain.Alloy
	var rawForm sql.NullString
	var extraIng sql.NullString
	var alloyType string
	if err := row.Scan(&a.ID, &a.Name, &alloyType, &rawForm, &extraIng); err != nil {
		if err == sql.ErrNoRows {
			return domain.Alloy{}, false
		}
		log.Printf("Error querying alloy by ID %s: %v", id, err)
		return domain.Alloy{}, false
	}
	if rawForm.Valid {
		raw := rawForm.String
		a.RawFormID = &raw
	}
	if extraIng.Valid {
		extra := extraIng.String
		a.ExtraIngredientID = &extra
	}
	if alloyType != "" {
		a.Type = domain.AlloyType(alloyType)
	}
	a.Ingredients = r.getIngredientsForAlloy(id)

	r.cacheLock.Lock()
	r.cache[id] = a
	r.cacheLock.Unlock()
	return a, true
}

// GetAllAlloys returns all alloys from the database.
func (r *Repository) GetAllAlloys() map[string]domain.Alloy {
	result := make(map[string]domain.Alloy)

	r.cacheLock.RLock()
	if len(r.cache) > 0 {
		for k, v := range r.cache {
			result[k] = v
		}
		r.cacheLock.RUnlock()
		return result
	}
	r.cacheLock.RUnlock()

	rows, err := r.db.Query(`SELECT id, name, type, raw_form_id, extra_ingredient_id FROM alloys`)
	if err != nil {
		log.Printf("Error querying all alloys: %v", err)
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var a domain.Alloy
		var rawForm sql.NullString
		var extraIng sql.NullString
		var alloyType string
		if err := rows.Scan(&a.ID, &a.Name, &alloyType, &rawForm, &extraIng); err != nil {
			log.Printf("Error scanning alloy row: %v", err)
			continue
		}
		if rawForm.Valid {
			raw := rawForm.String
			a.RawFormID = &raw
		}
		if extraIng.Valid {
			extra := extraIng.String
			a.ExtraIngredientID = &extra
		}
		if alloyType != "" {
			a.Type = domain.AlloyType(alloyType)
		}
		a.Ingredients = r.getIngredientsForAlloy(a.ID)

		r.cacheLock.Lock()
		r.cache[a.ID] = a
		r.cacheLock.Unlock()
		result[a.ID] = a
	}

	return result
}

// GetAlloyNameByID returns the human-readable name for a given ID.
func (r *Repository) GetAlloyNameByID(id string) string {
	alloy, ok := r.GetAlloyByID(id)
	if !ok {
		return ""
	}
	return alloy.Name
}

func (r *Repository) getIngredientsForAlloy(alloyID string) []domain.Ingredient {
	query := `
		SELECT ingredient_id, min_pct, max_pct
		FROM ingredients
		WHERE alloy_id = ?
	`
	rows, err := r.db.Query(query, alloyID)
	if err != nil {
		log.Printf("Error querying ingredients for %s: %v", alloyID, err)
		return nil
	}
	defer rows.Close()

	var list []domain.Ingredient
	for rows.Next() {
		var ing domain.Ingredient
		if err := rows.Scan(&ing.IngredientID, &ing.Min, &ing.Max); err != nil {
			log.Printf("Error scanning ingredient row for %s: %v", alloyID, err)
			continue
		}
		list = append(list, ing)
	}
	return list
}
