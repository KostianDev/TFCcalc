package domain

// AlloyRepository provides access to alloy definitions from a data source.
type AlloyRepository interface {
	GetAlloyByID(id string) (Alloy, bool)
	GetAllAlloys() map[string]Alloy
	GetAlloyNameByID(id string) string
}
