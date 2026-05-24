package ui

import (
	"fmt"
	"sort"
	"tfccalc/domain"
	"tfccalc/usecase/alloy"
)

// Tree formatting helpers without UI dependencies.

// calculationNode represents one node in the ingredient tree.
type calculationNode struct {
	ID           string             // Unique ID: "<alloyID>_lvl<level>_<counter>"
	AlloyID      string             // Alloy/material ID
	Name         string             // Human-readable name
	AmountMB     float64            // Amount in milli-buckets
	AmountIngots float64            // Amount in ingots (MB / 100)
	IsBaseMetal  bool               // True if this node is a base metal
	Children     []*calculationNode // Child nodes
}

// buildResultTreeRecursive builds the calculation tree for a given alloy.
func buildResultTreeRecursive(
	alloyID string,
	amountMB float64,
	percentages map[string]map[string]float64,
	visited map[string]int,
	level, maxLevel int,
	svc *alloy.Service,
) (*calculationNode, error) {
	if level > maxLevel {
		return nil, nil
	}

	// Generate a unique node ID for display and tests.
	nodeUID := fmt.Sprintf("%s_lvl%d_%d", alloyID, level, visited[alloyID])
	visited[alloyID]++

	alloyData, ok := svc.GetAlloyByID(alloyID)
	if !ok {
		return nil, fmt.Errorf("unknown alloy: %s", alloyID)
	}

	// Create the node for this alloy/material.
	node := &calculationNode{
		ID:           nodeUID,
		AlloyID:      alloyID,
		Name:         alloyData.Name,
		AmountMB:     amountMB,
		AmountIngots: amountMB / 100.0,
		IsBaseMetal:  alloyData.Type == domain.AlloyTypeBase,
	}

	idForIngredients := alloyID
	recipeSource := alloyData
	processed := false

	// For final_steel, include raw form and extra ingredient.
	if alloyData.Type == domain.AlloyTypeFinalSteel {
		if alloyData.RawFormID != nil {
			idForIngredients = *alloyData.RawFormID
		}
		recipeSource, ok = svc.GetAlloyByID(idForIngredients)
		if !ok {
			return nil, fmt.Errorf("raw_form %s not found", idForIngredients)
		}
		// Keep the final steel name, not the raw form name.
		node.Name = alloyData.Name

		// Recurse into the raw form.
		if alloyData.RawFormID != nil {
			if rawNode, err := buildResultTreeRecursive(
				idForIngredients, amountMB, percentages, visited, level+1, maxLevel, svc,
			); err == nil && rawNode != nil {
				node.Children = append(node.Children, rawNode)
			}
		}
		// Recurse into extra ingredient.
		if alloyData.ExtraIngredientID != nil {
			if extraNode, err := buildResultTreeRecursive(
				*alloyData.ExtraIngredientID, amountMB, percentages, visited, level+1, maxLevel, svc,
			); err == nil && extraNode != nil {
				node.Children = append(node.Children, extraNode)
			}
		}
		processed = true

	} else if alloyData.Type == domain.AlloyTypeProcessed && alloyID == "steel" {
		// Processed steel is 100% pig_iron.
		node.Name = alloyData.Name
		if pigNode, err := buildResultTreeRecursive(
			"pig_iron", amountMB, percentages, visited, level+1, maxLevel, svc,
		); err == nil && pigNode != nil {
			node.Children = append(node.Children, pigNode)
		}
		processed = true
	}

	// Standard case: alloys or raw_steel composed by percentage.
	if !processed && alloyData.Type != domain.AlloyTypeBase && len(recipeSource.Ingredients) > 0 {
		node.Name = recipeSource.Name

		// Merge default percentages with any overrides.
		defaultPerc, _ := svc.GetDefaultPercentages(idForIngredients)
		if userPerc, found := percentages[idForIngredients]; found && defaultPerc != nil {
			merged := make(map[string]float64)
			for k, v := range userPerc {
				merged[k] = v
			}
			for _, ing := range recipeSource.Ingredients {
				if _, exists := merged[ing.IngredientID]; !exists {
					merged[ing.IngredientID] = defaultPerc[ing.IngredientID]
				}
			}
			// Use merged values when valid.
			if valid, _ := svc.ValidatePercentages(idForIngredients, merged); valid {
				defaultPerc = merged
			}
		}

		// Fail if the final percentages are invalid.
		if valid, err := svc.ValidatePercentages(idForIngredients, defaultPerc); !valid {
			return nil, fmt.Errorf("invalid percentages for %s: %v", idForIngredients, err)
		}

		// Split amountMB among ingredients and recurse.
		for _, ing := range recipeSource.Ingredients {
			perc := defaultPerc[ing.IngredientID]
			childMB := amountMB * (perc / 100.0)
			if childMB < 1e-3 {
				continue
			}
			if childNode, err := buildResultTreeRecursive(
				ing.IngredientID, childMB, percentages, visited, level+1, maxLevel, svc,
			); err == nil && childNode != nil {
				node.Children = append(node.Children, childNode)
			}
		}
		// Sort children by name for stable output.
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].Name < node.Children[j].Name
		})
	}

	return node, nil
}

// lineInfo holds everything needed to render one ASCII tree line.
type lineInfo struct {
	PrefixParts []bool // true means ancestor was last at that depth
	IsLast      bool   // Is this node the last child at its level?
	Text        string // Node label, e.g. "Copper (221.25mB | 2.212Ing)"
}

// collectLines walks nodes and appends lineInfo entries.
func collectLines(nodes []*calculationNode, prefixParts []bool, out *[]lineInfo) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		lineText := fmt.Sprintf("%s (%.2fmB | %.3fIng)", node.Name, node.AmountMB, node.AmountIngots)
		*out = append(*out, lineInfo{
			PrefixParts: append(append([]bool{}, prefixParts...), isLast),
			IsLast:      isLast,
			Text:        lineText,
		})
		if len(node.Children) > 0 {
			collectLines(node.Children, append(prefixParts, isLast), out)
		}
	}
}

// formatHierarchy flattens one or more roots into lineInfo entries.
func formatHierarchy(roots []*calculationNode) []lineInfo {
	var lines []lineInfo
	if len(roots) == 0 {
		return lines
	}
	for idx, root := range roots {
		isLastRoot := idx == len(roots)-1
		lineText := fmt.Sprintf("%s (%.2fmB | %.3fIng)", root.Name, root.AmountMB, root.AmountIngots)
		lines = append(lines, lineInfo{
			PrefixParts: []bool{isLastRoot}, // Top level uses a single boolean
			IsLast:      isLastRoot,
			Text:        lineText,
		})
		if len(root.Children) > 0 {
			collectLines(root.Children, []bool{isLastRoot}, &lines)
		}
	}
	return lines
}
