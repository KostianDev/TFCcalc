package ui

import (
	"fmt"
	"math"
	"tfccalc/domain"
	"tfccalc/usecase/alloy"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Helpers for percentage sliders and accordion items.

// percentageControl holds widgets and bounds for one ingredient percentage.
type percentageControl struct {
	alloyID     string
	ingredient  string
	min         float64
	max         float64
	orderIndex  int
	slider      *widget.Slider
	valueLabel  *widget.Label
	lockToggle  *widget.Check
	lockedValue float64
	defaultVal  float64
	userTouched bool
}

// createPercentageInputsForAlloy builds sliders for one alloy's ingredients.
func createPercentageInputsForAlloy(alloyID string, svc *alloy.Service) (fyne.CanvasObject, error) {
	alloyData, ok := svc.GetAlloyByID(alloyID)
	if !ok || len(alloyData.Ingredients) == 0 {
		lbl := widget.NewLabel("  (No configurable ingredients)")
		lbl.Wrapping = fyne.TextWrapWord
		return lbl, nil
	}

	vbox := container.NewVBox()
	currentMap := make(map[string]*percentageControl)
	alloyPercentageControls[alloyID] = currentMap
	if alloyPercentageWarnings == nil {
		alloyPercentageWarnings = make(map[string]*widget.Label)
	}
	if alloyPercentageUpdating == nil {
		alloyPercentageUpdating = make(map[string]bool)
	}

	defaultPerc, _ := svc.GetDefaultPercentages(alloyID)

	resetButton := widget.NewButton("Reset to Defaults", func() {
		if defaultPerc == nil {
			return
		}
		alloyPercentageUpdating[alloyID] = true
		for _, ctl := range currentMap {
			if val, ok := defaultPerc[ctl.ingredient]; ok {
				ctl.slider.SetValue(val)
				ctl.valueLabel.SetText(fmt.Sprintf("%.1f%%", val))
				ctl.valueLabel.Refresh()
				ctl.defaultVal = val
			}
			ctl.lockToggle.SetChecked(false)
			ctl.slider.Enable()
			ctl.userTouched = false
			ctl.slider.Refresh()
		}
		if warning := alloyPercentageWarnings[alloyID]; warning != nil {
			warning.SetText("")
		}
		alloyPercentageUpdating[alloyID] = false
	})
	vbox.Add(resetButton)

	for idx, ing := range alloyData.Ingredients {
		ingredientID := ing.IngredientID
		ingName := svc.GetAlloyNameByID(ingredientID)
		label := widget.NewLabel(fmt.Sprintf("%s [%.0f-%.0f%%]:", ingName, ing.Min, ing.Max))
		label.Wrapping = fyne.TextWrapWord

		slider := widget.NewSlider(ing.Min, ing.Max)
		slider.Step = 0.1
		valueLabel := widget.NewLabel("0.0%")
		valueLabel.Alignment = fyne.TextAlignTrailing

		lockToggle := widget.NewCheck("Lock", nil)

		control := &percentageControl{
			alloyID:    alloyID,
			ingredient: ingredientID,
			min:        ing.Min,
			max:        ing.Max,
			orderIndex: idx,
			slider:     slider,
			valueLabel: valueLabel,
			lockToggle: lockToggle,
		}
		currentMap[ingredientID] = control
		ctl := control

		if defaultPerc != nil {
			if val, found := defaultPerc[ingredientID]; found {
				slider.SetValue(val)
				valueLabel.SetText(fmt.Sprintf("%.1f%%", val))
				valueLabel.Refresh()
				ctl.defaultVal = val
			}
		}

		lockToggle.OnChanged = func(locked bool) {
			ctl.lockedValue = ctl.slider.Value
			if locked {
				ctl.slider.Disable()
			} else {
				ctl.slider.Enable()
			}
		}

		slider.OnChanged = func(val float64) {
			if !alloyPercentageUpdating[alloyID] {
				ctl.userTouched = true
			}
			if alloyPercentageUpdating[alloyID] {
				return
			}
			alloyPercentageUpdating[alloyID] = true
			clamped, warningText := rebalanceAlloyPercentages(alloyID, ingredientID, val)
			warning := alloyPercentageWarnings[alloyID]
			if warning != nil {
				if warningText != "" {
					warning.SetText(warningText)
				} else if clamped {
					warning.SetText("Adjusted to keep total at 100% within limits.")
				} else {
					warning.SetText("")
				}
			}
			alloyPercentageUpdating[alloyID] = false
		}

		vbox.Add(container.NewGridWithColumns(4, label, slider, valueLabel, lockToggle))
	}

	warningLabel := widget.NewLabel("")
	warningLabel.Wrapping = fyne.TextWrapWord
	alloyPercentageWarnings[alloyID] = warningLabel
	vbox.Add(warningLabel)
	return vbox, nil
}

func rebalanceAlloyPercentages(alloyID, changedID string, requested float64) (bool, string) {
	controls := alloyPercentageControls[alloyID]
	if len(controls) == 0 {
		return false, ""
	}
	changed, ok := controls[changedID]
	if !ok {
		return false, ""
	}

	requested = clamp(requested, changed.min, changed.max)

	fixedSum := 0.0
	adjustable := make([]*percentageControl, 0)
	for id, ctl := range controls {
		if id == changedID {
			continue
		}
		if ctl.lockToggle.Checked || ctl.userTouched {
			fixedSum += ctl.slider.Value
			continue
		}
		adjustable = append(adjustable, ctl)
	}

	minOthers := fixedSum
	maxOthers := fixedSum
	for _, ctl := range adjustable {
		minOthers += ctl.min
		maxOthers += ctl.max
	}

	minAllowed := math.Max(changed.min, 100.0-maxOthers)
	maxAllowed := math.Min(changed.max, 100.0-minOthers)
	if minAllowed > maxAllowed {
		minAllowed = maxAllowed
	}
	clampedValue := clamp(requested, minAllowed, maxAllowed)
	clamped := math.Abs(clampedValue-requested) > 1e-6

	changed.slider.SetValue(clampedValue)
	changed.valueLabel.SetText(fmt.Sprintf("%.1f%%", clampedValue))
	changed.valueLabel.Refresh()
	changed.slider.Refresh()

	if len(adjustable) == 0 {
		return clamped, ""
	}

	minAdjustable := 0.0
	maxAdjustable := 0.0
	for _, ctl := range adjustable {
		minAdjustable += ctl.min
		maxAdjustable += ctl.max
	}

	totalMin := fixedSum + clampedValue + minAdjustable
	totalMax := fixedSum + clampedValue + maxAdjustable
	if totalMin > 100.0 {
		for _, ctl := range adjustable {
			ctl.slider.SetValue(ctl.min)
			ctl.valueLabel.SetText(fmt.Sprintf("%.1f%%", ctl.min))
			ctl.valueLabel.Refresh()
			ctl.slider.Refresh()
		}
		return clamped, "Fixed sliders are above 100%. Adjust touched values to continue."
	}
	if totalMax < 100.0 {
		for _, ctl := range adjustable {
			ctl.slider.SetValue(ctl.max)
			ctl.valueLabel.SetText(fmt.Sprintf("%.1f%%", ctl.max))
			ctl.valueLabel.Refresh()
			ctl.slider.Refresh()
		}
		return clamped, "Fixed sliders are below 100%. Adjust touched values to continue."
	}

	targetOthersSum := 100.0 - fixedSum - clampedValue
	values := make(map[*percentageControl]float64, len(adjustable))
	currentSum := 0.0
	for _, ctl := range adjustable {
		values[ctl] = ctl.slider.Value
		currentSum += ctl.slider.Value
	}

	diff := targetOthersSum - currentSum
	remaining := append([]*percentageControl{}, adjustable...)
	for len(remaining) > 0 && math.Abs(diff) > 1e-6 {
		step := diff / float64(len(remaining))
		next := make([]*percentageControl, 0, len(remaining))
		for _, ctl := range remaining {
			current := values[ctl]
			proposed := current + step
			if proposed < ctl.min {
				values[ctl] = ctl.min
				diff -= (ctl.min - current)
				continue
			}
			if proposed > ctl.max {
				values[ctl] = ctl.max
				diff -= (ctl.max - current)
				continue
			}
			values[ctl] = proposed
			diff -= step
			next = append(next, ctl)
		}
		remaining = next
	}

	for ctl, val := range values {
		ctl.slider.SetValue(val)
		ctl.valueLabel.SetText(fmt.Sprintf("%.1f%%", val))
		ctl.valueLabel.Refresh()
		ctl.slider.Refresh()
	}

	return clamped, ""
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// buildAccordionItemsRecursive walks the alloy-to-ingredients graph and appends
// an AccordionItem for each alloy (or raw form) with configurable ingredients.
func buildAccordionItemsRecursive(alloyID string, acc *widget.Accordion, visited map[string]bool, svc *alloy.Service) {
	if visited[alloyID] {
		return
	}
	visited[alloyID] = true

	alloyData, ok := svc.GetAlloyByID(alloyID)
	if !ok {
		return
	}
	idForInputs := alloyID
	if alloyData.Type == domain.AlloyTypeFinalSteel && alloyData.RawFormID != nil {
		idForInputs = *alloyData.RawFormID
	}

	currentAlloy, ok := svc.GetAlloyByID(idForInputs)
	if !ok {
		return
	}
	// If this alloy/form has ingredients, add a "Configure: <Name>" item.
	if len(currentAlloy.Ingredients) > 0 {
		content, err := createPercentageInputsForAlloy(idForInputs, svc)
		if err != nil {
			lbl := widget.NewLabel(fmt.Sprintf("Error loading inputs: %v", err))
			lbl.Wrapping = fyne.TextWrapWord
			content = lbl
		}
		item := widget.NewAccordionItem(fmt.Sprintf("Configure: %s", currentAlloy.Name), content)
		acc.Append(item)

		// Recurse into ingredients that are alloys or raw_steel.
		for _, ing := range currentAlloy.Ingredients {
			ingAlloy, ok2 := svc.GetAlloyByID(ing.IngredientID)
			if !ok2 {
				continue
			}
			nextID := ing.IngredientID
			if ingAlloy.Type == domain.AlloyTypeFinalSteel && ingAlloy.RawFormID != nil {
				nextID = *ingAlloy.RawFormID
			}
			nextAlloy, ok3 := svc.GetAlloyByID(nextID)
			if ok3 && (nextAlloy.Type == domain.AlloyTypeAlloy || nextAlloy.Type == domain.AlloyTypeRawSteel) && len(nextAlloy.Ingredients) > 0 {
				buildAccordionItemsRecursive(nextID, acc, visited, svc)
			}
		}
	} else if currentAlloy.Type == domain.AlloyTypeAlloy || currentAlloy.Type == domain.AlloyTypeRawSteel {
		// If it is a leaf alloy, still show a "Configure: <Name> (No ingredients)" label.
		lbl := widget.NewLabel(" (No configurable ingredients)")
		lbl.Wrapping = fyne.TextWrapWord
		acc.Append(widget.NewAccordionItem(fmt.Sprintf("Configure: %s", currentAlloy.Name), lbl))
	}
}
