// ui/percent_inputs.go
package ui

import (
	"fmt"
	"tfccalc/calculator"
	"tfccalc/data"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
)

//
// Functions for creating percentage‐input fields and populating the accordion:
// - createPercentageInputsForAlloy
// - buildAccordionItemsRecursive
//

// createPercentageInputsForAlloy builds a container (VBox or Label) showing Label+Entry
// pairs for each ingredient of the given alloyID. If there are no ingredients, it returns
// a simple Label saying “(No configurable ingredients).”
func createPercentageInputsForAlloy(alloyID string) (fyne.CanvasObject, error) {
	alloy, ok := data.GetAlloyByID(alloyID)
	if !ok || len(alloy.Ingredients) == 0 {
		lbl := widget.NewLabel("  (No configurable ingredients)")
		lbl.Wrapping = fyne.TextWrapWord
		return lbl, nil
	}

	vbox := container.NewVBox()
	currentMap := make(map[string]*widget.Entry)
	alloyPercentageEntries[alloyID] = currentMap

	defaultPerc, _ := calculator.GetDefaultPercentages(alloyID)
	for _, ing := range alloy.Ingredients {
		ingName := data.GetAlloyNameByID(ing.IngredientID)
		label := widget.NewLabel(fmt.Sprintf("%s [%.0f–%.0f%%]:", ingName, ing.Min, ing.Max))
		label.Wrapping = fyne.TextWrapWord

		entry := widget.NewEntry()
		entry.Validator = validation.NewRegexp(`^\d+(\.\d+)?$`, "Number")
		if defaultPerc != nil {
			if val, found := defaultPerc[ing.IngredientID]; found {
				entry.PlaceHolder = fmt.Sprintf("%.1f", val)
			} else {
				entry.PlaceHolder = "?"
			}
		}
		entry.Wrapping = fyne.TextTruncate

		currentMap[ing.IngredientID] = entry
		vbox.Add(container.NewGridWithColumns(2, label, entry))
	}
	return vbox, nil
}

// buildAccordionItemsRecursive walks the alloy → ingredients graph and appends an
// AccordionItem for every alloy (or raw form) that has configurable ingredients.
// It uses visited to avoid infinite cycles.
func buildAccordionItemsRecursive(alloyID string, acc *widget.Accordion, visited map[string]bool) {
	if visited[alloyID] {
		return
	}
	visited[alloyID] = true

	alloy, ok := data.GetAlloyByID(alloyID)
	if !ok {
		return
	}
	idForInputs := alloyID
	if alloy.Type == "final_steel" {
		idForInputs = alloy.RawFormID.String
	}

	currentAlloy, ok := data.GetAlloyByID(idForInputs)
	if !ok {
		return
	}
	// If this alloy/form has ingredients, add a “Configure: <Name>” item.
	if len(currentAlloy.Ingredients) > 0 {
		content, err := createPercentageInputsForAlloy(idForInputs)
		if err != nil {
			lbl := widget.NewLabel(fmt.Sprintf("Error loading inputs: %v", err))
			lbl.Wrapping = fyne.TextWrapWord
			content = lbl
		}
		item := widget.NewAccordionItem(fmt.Sprintf("Configure: %s", currentAlloy.Name), content)
		acc.Append(item)

		// Recurse into each ingredient that is itself an alloy or raw_steel.
		for _, ing := range currentAlloy.Ingredients {
			ingAlloy, ok2 := data.GetAlloyByID(ing.IngredientID)
			if !ok2 {
				continue
			}
			nextID := ing.IngredientID
			if ingAlloy.Type == "final_steel" {
				nextID = ingAlloy.RawFormID.String
			}
			nextAlloy, ok3 := data.GetAlloyByID(nextID)
			if ok3 && (nextAlloy.Type == "alloy" || nextAlloy.Type == "raw_steel") && len(nextAlloy.Ingredients) > 0 {
				buildAccordionItemsRecursive(nextID, acc, visited)
			}
		}
	} else if currentAlloy.Type == "alloy" || currentAlloy.Type == "raw_steel" {
		// If it’s a leaf alloy, still show a “Configure: <Name> (No ingredients)” label.
		lbl := widget.NewLabel(" (No configurable ingredients)")
		lbl.Wrapping = fyne.TextWrapWord
		acc.Append(widget.NewAccordionItem(fmt.Sprintf("Configure: %s", currentAlloy.Name), lbl))
	}
}
