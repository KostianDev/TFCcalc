// tfccalc/data/alloys.go
package data

import "fmt"

// GetAlloyByID returns (AlloyInfo, true) if found, or (zero, false) otherwise.
// Internally calls dbGetAlloyByID from db.go.
func GetAlloyByID(id string) (AlloyInfo, bool) {
	return dbGetAlloyByID(id)
}

// GetAlloyNameByID returns the human-readable name for a given ID, or "Unknown[ID]" if not found.
func GetAlloyNameByID(id string) string {
	a, ok := dbGetAlloyByID(id)
	if !ok {
		return fmt.Sprintf("Unknown[%s]", id)
	}
	return a.Name
}

// GetAllAlloys returns a map[id]→AlloyInfo for all alloys/materials.
// Internally calls dbGetAllAlloys from db.go.
func GetAllAlloys() map[string]AlloyInfo {
	return dbGetAllAlloys()
}
