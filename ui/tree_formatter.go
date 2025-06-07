package ui

import (
	"fmt"
	"sort"
	"tfccalc/calculator"
	"tfccalc/data"
)

//
// This file contains “pure” logic for building the calculation tree and formatting it
// into a slice of lineInfo structs. It does not create any Fyne widgets.
//
// - calculationNode
// - buildResultTreeRecursive
// - lineInfo, collectLines, formatHierarchy
//

// calculationNode represents one node in the ingredient‐breakdown tree.
type calculationNode struct {
	ID           string             // Unique ID: "<alloyID>_lvl<level>_<counter>"
	AlloyID      string             // Underlying alloy/material ID
	Name         string             // Human‐readable name
	AmountMB     float64            // Amount in milli‐Buckets
	AmountIngots float64            // Amount in Ingots (MB / 100)
	IsBaseMetal  bool               // True if this node is a raw base metal
	Children     []*calculationNode // Child nodes (ingredients)
}

// buildResultTreeRecursive builds the calculation tree for a given alloy.
// Parameters:
//   - alloyID: ID of the alloy/material to expand.
//   - amountMB: requested amount in milli‐Buckets.
//   - percentages: map[alloyID]→map[ingredientID]→percentage override.
//   - visited: map to track how many times each alloyID has been visited (to avoid infinite loops).
//   - level, maxLevel: current depth and maximum depth to recurse.
func buildResultTreeRecursive(
	alloyID string,
	amountMB float64,
	percentages map[string]map[string]float64,
	visited map[string]int,
	level, maxLevel int,
) (*calculationNode, error) {
	if level > maxLevel {
		return nil, nil
	}

	// Generate a unique node ID so that we can display it or test it later.
	nodeUID := fmt.Sprintf("%s_lvl%d_%d", alloyID, level, visited[alloyID])
	visited[alloyID]++

	alloyData, ok := data.GetAlloyByID(alloyID)
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
		IsBaseMetal:  alloyData.Type == "base",
	}

	idForIngredients := alloyID
	recipeSource := alloyData
	processed := false

	// 1) If this is a final_steel alloy, first add its raw form and extra ingredient.
	if alloyData.Type == "final_steel" {
		idForIngredients = alloyData.RawFormID.String
		recipeSource, ok = data.GetAlloyByID(idForIngredients)
		if !ok {
			return nil, fmt.Errorf("raw_form %s not found", idForIngredients)
		}
		// Keep the node’s Name as the final steel name, not the raw form.
		node.Name = alloyData.Name

		// Recurse into the raw form
		if alloyData.RawFormID.Valid {
			if rawNode, err := buildResultTreeRecursive(
				idForIngredients, amountMB, percentages, visited, level+1, maxLevel,
			); err == nil && rawNode != nil {
				node.Children = append(node.Children, rawNode)
			}
		}
		// Recurse into any extra ingredient
		if alloyData.ExtraIngredientID.Valid {
			if extraNode, err := buildResultTreeRecursive(
				alloyData.ExtraIngredientID.String, amountMB, percentages, visited, level+1, maxLevel,
			); err == nil && extraNode != nil {
				node.Children = append(node.Children, extraNode)
			}
		}
		processed = true

	} else if alloyData.Type == "processed" && alloyID == "steel" {
		// 2) If this is the processed steel, it is 100% pig_iron.
		node.Name = alloyData.Name
		if pigNode, err := buildResultTreeRecursive(
			"pig_iron", amountMB, percentages, visited, level+1, maxLevel,
		); err == nil && pigNode != nil {
			node.Children = append(node.Children, pigNode)
		}
		processed = true
	}

	// 3) Standard case: alloys or raw_steel composed of ingredients by percentage.
	if !processed && alloyData.Type != "base" && len(recipeSource.Ingredients) > 0 {
		node.Name = recipeSource.Name

		// Get default percentages and merge in any user overrides.
		defaultPerc, _ := calculator.GetDefaultPercentages(idForIngredients)
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
			// If the merged percentages are valid, use them.
			if valid, _ := calculator.ValidatePercentages(idForIngredients, merged); valid {
				defaultPerc = merged
			}
		}

		// If the final defaultPerc map is invalid, return an error.
		if valid, err := calculator.ValidatePercentages(idForIngredients, defaultPerc); !valid {
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
				ing.IngredientID, childMB, percentages, visited, level+1, maxLevel,
			); err == nil && childNode != nil {
				node.Children = append(node.Children, childNode)
			}
		}
		// Sort children alphabetically by Name to keep output stable.
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].Name < node.Children[j].Name
		})
	}

	return node, nil
}

// lineInfo holds everything needed to render one ASCII‐tree line:
//
//   - PrefixParts: for each ancestor level, true=that ancestor was the last child, so we print spaces.
//   - IsLast: is this node the last among its siblings (so we choose “└── ” vs. “├── ”).
//   - Text: e.g. “Bismuth Bronze (250.00mB | 2.500Ing)”.
type lineInfo struct {
	PrefixParts []bool // PrefixParts[i] == true ⇒ at depth i, ancestor was last ⇒ print spaces
	IsLast      bool   // Is this node the last child at its level?
	Text        string // Node label, e.g. “Copper (221.25mB | 2.212Ing)”
}

// collectLines recursively walks nodes and appends lineInfo entries.
// prefixParts is passed down so that each child inherits which ancestors were “last”.
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

// formatHierarchy takes one or more root nodes and returns a flat slice of lineInfo
// representing the entire forest. This can then be passed to RenderLines.
func formatHierarchy(roots []*calculationNode) []lineInfo {
	var lines []lineInfo
	if len(roots) == 0 {
		return lines
	}
	for idx, root := range roots {
		isLastRoot := idx == len(roots)-1
		lineText := fmt.Sprintf("%s (%.2fmB | %.3fIng)", root.Name, root.AmountMB, root.AmountIngots)
		lines = append(lines, lineInfo{
			PrefixParts: []bool{isLastRoot}, // top‐level depth uses only one boolean
			IsLast:      isLastRoot,
			Text:        lineText,
		})
		if len(root.Children) > 0 {
			collectLines(root.Children, []bool{isLastRoot}, &lines)
		}
	}
	return lines
}
