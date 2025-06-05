package ui

import (
	"fmt"
	"image/color"
	"log"
	"sort"
	"strconv"
	"strings"
	"tfccalc/calculator"
	"tfccalc/data"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
)

// --- Data structures & helpers for building the colored ASCII tree ---

// calculationNode represents a node in the ingredient breakdown tree.
type calculationNode struct {
	ID           string
	AlloyID      string
	Name         string
	AmountMB     float64
	AmountIngots float64
	IsBaseMetal  bool
	Children     []*calculationNode
}

// buildResultTreeRecursive constructs the calculation tree for an alloy.
func buildResultTreeRecursive(alloyID string, amountMB float64, percentages map[string]map[string]float64, visited map[string]int, level, maxLevel int) (*calculationNode, error) {
	if level > maxLevel {
		return nil, nil
	}
	nodeUID := fmt.Sprintf("%s_lvl%d_%d", alloyID, level, visited[alloyID])
	visited[alloyID]++
	alloyData, ok := data.GetAlloyByID(alloyID)
	if !ok {
		return nil, fmt.Errorf("unknown alloy: %s", alloyID)
	}
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

	// Handle final_steel
	if alloyData.Type == "final_steel" {
		idForIngredients = alloyData.RawFormID.String
		recipeSource, ok = data.GetAlloyByID(idForIngredients)
		if !ok {
			return nil, fmt.Errorf("raw_form %s not found", idForIngredients)
		}
		node.Name = alloyData.Name
		if alloyData.RawFormID.Valid {
			if rawNode, err := buildResultTreeRecursive(idForIngredients, amountMB, percentages, visited, level+1, maxLevel); err == nil && rawNode != nil {
				node.Children = append(node.Children, rawNode)
			}
		}
		if alloyData.ExtraIngredientID.Valid {
			if extraNode, err := buildResultTreeRecursive(alloyData.ExtraIngredientID.String, amountMB, percentages, visited, level+1, maxLevel); err == nil && extraNode != nil {
				node.Children = append(node.Children, extraNode)
			}
		}
		processed = true
	} else if alloyData.Type == "processed" && alloyID == "steel" {
		// Steel → pig_iron
		node.Name = alloyData.Name
		if pigNode, err := buildResultTreeRecursive("pig_iron", amountMB, percentages, visited, level+1, maxLevel); err == nil && pigNode != nil {
			node.Children = append(node.Children, pigNode)
		}
		processed = true
	}

	// Standard alloy/raw_steel
	if !processed && alloyData.Type != "base" && len(recipeSource.Ingredients) > 0 {
		node.Name = recipeSource.Name
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
			if valid, _ := calculator.ValidatePercentages(idForIngredients, merged); valid {
				defaultPerc = merged
			}
		}
		if valid, err := calculator.ValidatePercentages(idForIngredients, defaultPerc); !valid {
			return nil, fmt.Errorf("invalid percentages for %s: %v", idForIngredients, err)
		}
		for _, ing := range recipeSource.Ingredients {
			perc := defaultPerc[ing.IngredientID]
			childMB := amountMB * (perc / 100.0)
			if childMB < 0.001 {
				continue
			}
			if childNode, err := buildResultTreeRecursive(ing.IngredientID, childMB, percentages, visited, level+1, maxLevel); err == nil && childNode != nil {
				node.Children = append(node.Children, childNode)
			}
		}
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].Name < node.Children[j].Name
		})
	}

	return node, nil
}

// lineInfo holds the data needed to render one line of the ASCII tree.
type lineInfo struct {
	PrefixParts []bool // prefixParts[i]==true → at depth i, ancestor was last → render spaces
	IsLast      bool   // is this node the last among its siblings?
	Text        string // "Name (mB | Ing)"
}

// collectLines recursively traverses the tree and appends lineInfo entries.
func collectLines(nodes []*calculationNode, prefixParts []bool, out *[]lineInfo) {
	for i, node := range nodes {
		isLast := (i == len(nodes)-1)
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

// formatHierarchy produces a slice of lineInfo for the entire forest.
func formatHierarchy(roots []*calculationNode) []lineInfo {
	var lines []lineInfo
	if len(roots) == 0 {
		return lines
	}
	for i, root := range roots {
		isLastRoot := (i == len(roots)-1)
		lineText := fmt.Sprintf("%s (%.2fmB | %.3fIng)", root.Name, root.AmountMB, root.AmountIngots)
		lines = append(lines, lineInfo{
			PrefixParts: []bool{isLastRoot},
			IsLast:      isLastRoot,
			Text:        lineText,
		})
		if len(root.Children) > 0 {
			collectLines(root.Children, []bool{isLastRoot}, &lines)
		}
	}
	return lines
}

// --- UI variables & initialization ---

var (
	alloyNames             []string                                    // Names of available alloys.
	alloyIDs               = map[string]string{}                       // Maps alloy name → ID.
	alloyPercentageEntries = make(map[string]map[string]*widget.Entry) // User‐entered percentage fields.
	percentageAccordion    *widget.Accordion                           // Accordion for percentage inputs.
	hierarchyContainer     *fyne.Container                             // VBox container to show colored ASCII‐tree lines.
	summaryTable           *widget.Table                               // Table widget for final summary.
	summaryData            [][]string                                  // Data rows for summary table.
	currentAlloyID         string                                      // ID of the currently selected alloy.
	amountEntry            *widget.Entry                               // Entry for the desired amount.
	modeRadio              *widget.RadioGroup                          // Radio group to select mB or Ingots.
	statusLabel            *widget.Label                               // Label for status messages.
)

// createPercentageInputsForAlloy builds entry fields for adjustable percentages.
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

// buildAccordionItemsRecursive populates the accordion with percentage inputs.
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
	if len(currentAlloy.Ingredients) > 0 {
		content, err := createPercentageInputsForAlloy(idForInputs)
		if err != nil {
			lbl := widget.NewLabel(fmt.Sprintf("Error loading inputs: %v", err))
			lbl.Wrapping = fyne.TextWrapWord
			content = lbl
		}
		item := widget.NewAccordionItem(fmt.Sprintf("Configure: %s", currentAlloy.Name), content)
		acc.Append(item)
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
		lbl := widget.NewLabel(" (No configurable ingredients)")
		lbl.Wrapping = fyne.TextWrapWord
		acc.Append(widget.NewAccordionItem(fmt.Sprintf("Configure: %s", currentAlloy.Name), lbl))
	}
}

// BuildUI constructs and returns the main window with colored ASCII‐tree hierarchy.
func BuildUI(app fyne.App) fyne.Window {
	// Color palette: distinct color per depth (repeats if depth exceeds palette).
	palette := []color.Color{
		color.RGBA{R: 255, G: 102, B: 102, A: 255}, // Light Red
		color.RGBA{R: 102, G: 255, B: 102, A: 255}, // Light Green
		color.RGBA{R: 102, G: 178, B: 255, A: 255}, // Light Blue
		color.RGBA{R: 255, G: 255, B: 102, A: 255}, // Light Yellow
		color.RGBA{R: 255, G: 153, B: 255, A: 255}, // Light Pink
		color.RGBA{R: 153, G: 255, B: 255, A: 255}, // Light Cyan
	}

	// Load icon if available
	resIcon, err := fyne.LoadResourceFromPath("./assets/tfc_icon.png")
	if err != nil {
		log.Println("Error loading icon:", err)
	}

	win := app.NewWindow("TFC Alloy Calculator")
	win.SetIcon(resIcon)
	win.SetMaster()

	// Prepare alloy selector
	alloyNames = []string{}
	alloyIDs = make(map[string]string)
	for id, alloyData := range data.GetAllAlloys() {
		if alloyData.Type == "alloy" || alloyData.Type == "final_steel" {
			alloyNames = append(alloyNames, alloyData.Name)
			alloyIDs[alloyData.Name] = id
		}
	}
	sort.Strings(alloyNames)

	alloySelector := widget.NewSelect(alloyNames, func(name string) {
		newID := alloyIDs[name]
		if currentAlloyID == newID {
			return
		}
		currentAlloyID = newID
		alloyPercentageEntries = make(map[string]map[string]*widget.Entry)
		percentageAccordion.Items = nil
		visited := make(map[string]bool)
		startID := currentAlloyID
		if alloy, ok := data.GetAlloyByID(currentAlloyID); ok && alloy.Type == "final_steel" {
			startID = alloy.RawFormID.String
		}
		buildAccordionItemsRecursive(startID, percentageAccordion, visited)
		percentageAccordion.Refresh()
		if len(percentageAccordion.Items) > 0 {
			percentageAccordion.Open(0)
		} else {
			noItem := widget.NewAccordionItem("Percentage Configuration", widget.NewLabel("No configurable ingredients for this alloy."))
			noItem.Open = true
			percentageAccordion.Append(noItem)
			percentageAccordion.Refresh()
		}
		// Clear hierarchy and summary on alloy change
		hierarchyContainer.Objects = nil
		hierarchyContainer.Refresh()
		summaryData = [][]string{{"Material", "mB", "Ingots"}}
		summaryTable.Refresh()
		statusLabel.SetText("Select amount and mode, then press Calculate.")
	})
	alloySelector.PlaceHolder = "Select alloy..."

	// Amount entry
	amountEntry = widget.NewEntry()
	amountEntry.PlaceHolder = "Amount..."
	amountEntry.Validator = validation.NewRegexp(`^\d+(\.\d+)?$`, "Number > 0")

	// Mode radio
	modeRadio = widget.NewRadioGroup([]string{"mB", "Ingots"}, nil)
	modeRadio.Horizontal = true
	modeRadio.SetSelected("Ingots")

	// Status label
	statusLabel = widget.NewLabel("Enter data and press Calculate.")
	statusLabel.Wrapping = fyne.TextWrapWord

	// Percentage accordion
	percentageAccordion = widget.NewAccordion()
	accordionScroll := container.NewVScroll(percentageAccordion)
	accordionScroll.SetMinSize(fyne.NewSize(0, 200))

	// Hierarchy container (VBox) and Scroll
	hierarchyContainer = container.NewVBox() // Holds canvas.Text lines
	hierarchyScroll := container.NewScroll(hierarchyContainer)
	hierarchyScroll.SetMinSize(fyne.NewSize(0, 300))

	// Summary table initialization
	summaryData = [][]string{{"Material", "mB", "Ingots"}}
	summaryTable = widget.NewTable(
		func() (int, int) {
			return len(summaryData), 3
		},
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.Alignment = fyne.TextAlignLeading
			return container.NewPadded(lbl)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			cont := cell.(*fyne.Container)
			lbl := cont.Objects[0].(*widget.Label)
			if id.Row < len(summaryData) && id.Col < len(summaryData[id.Row]) {
				lbl.SetText(summaryData[id.Row][id.Col])
				if id.Row == 0 {
					lbl.TextStyle.Bold = true
					lbl.Alignment = fyne.TextAlignCenter
				} else {
					lbl.TextStyle.Bold = false
					if id.Col == 0 {
						lbl.Alignment = fyne.TextAlignLeading
					} else {
						lbl.Alignment = fyne.TextAlignTrailing
					}
				}
			} else {
				lbl.SetText("")
			}
		},
	)
	summaryTable.SetColumnWidth(0, 200)
	summaryTable.SetColumnWidth(1, 100)
	summaryTable.SetColumnWidth(2, 100)

	// Calculate button
	calcButton := widget.NewButton("Calculate", func() {
		statusLabel.SetText("Calculating...")
		selected := currentAlloyID
		if selected == "" {
			statusLabel.SetText("Error: Alloy not selected.")
			return
		}
		amtStr := amountEntry.Text
		amt, err := strconv.ParseFloat(amtStr, 64)
		if err != nil || amt <= 0 {
			statusLabel.SetText("Error: Enter a valid positive amount.")
			return
		}
		mode := modeRadio.Selected
		if mode == "" {
			statusLabel.SetText("Error: Select mode (mB or Ingots).")
			return
		}

		// Collect user percentages
		userPercs := make(map[string]map[string]float64)
		var validationErrors []string
		for alloyID, entryMap := range alloyPercentageEntries {
			tmp := make(map[string]float64)
			useCustom := false
			defaultPerc, _ := calculator.GetDefaultPercentages(alloyID)
			alloyInfo, _ := data.GetAlloyByID(alloyID)
			for ingID, entry := range entryMap {
				if entry.Text != "" {
					val, err2 := strconv.ParseFloat(entry.Text, 64)
					if err2 != nil {
						validationErrors = append(
							validationErrors,
							fmt.Sprintf("Invalid %% for %s in %s",
								data.GetAlloyNameByID(ingID),
								data.GetAlloyNameByID(alloyID),
							),
						)
						continue
					}
					tmp[ingID] = val
					useCustom = true
				}
			}
			if useCustom || len(alloyInfo.Ingredients) > 0 {
				finalPerc := make(map[string]float64)
				for k, v := range tmp {
					finalPerc[k] = v
				}
				if defaultPerc != nil {
					for _, ing := range alloyInfo.Ingredients {
						if _, exists := finalPerc[ing.IngredientID]; !exists {
							if defv, ok := defaultPerc[ing.IngredientID]; ok {
								finalPerc[ing.IngredientID] = defv
							} else {
								validationErrors = append(
									validationErrors,
									fmt.Sprintf("No default for %s in %s",
										data.GetAlloyNameByID(ing.IngredientID),
										data.GetAlloyNameByID(alloyID),
									),
								)
							}
						}
					}
				}
				valid, errv := calculator.ValidatePercentages(alloyID, finalPerc)
				if !valid {
					validationErrors = append(
						validationErrors,
						fmt.Sprintf("Error in %% for %s: %v",
							data.GetAlloyNameByID(alloyID),
							errv,
						),
					)
				} else if len(finalPerc) > 0 {
					userPercs[alloyID] = finalPerc
				}
			}
		}
		if len(validationErrors) > 0 {
			statusLabel.SetText("Percentage errors:\n- " + strings.Join(validationErrors, "\n- "))
			return
		}

		var percMap map[string]map[string]float64
		if len(userPercs) > 0 {
			percMap = userPercs
		}
		finalMB, _, errCalc := calculator.CalculateRequirements(selected, amt, mode, percMap)
		if errCalc != nil {
			statusLabel.SetText(fmt.Sprintf("Calculation error:\n%v", errCalc))
			hierarchyContainer.Objects = nil
			hierarchyContainer.Refresh()
			summaryData = [][]string{{"Material", "mB", "Ingots"}}
			summaryTable.Refresh()
			return
		}

		// Build calculation tree
		rootMB := amt
		if mode == "Ingots" {
			rootMB = amt * 100.0
		}
		rootNode, errTree := buildResultTreeRecursive(selected, rootMB, percMap, make(map[string]int), 0, 5)
		if errTree != nil {
			statusLabel.SetText(fmt.Sprintf("Tree build error: %v", errTree))
			hierarchyContainer.Objects = nil
			hierarchyContainer.Refresh()
		} else if rootNode != nil {
			// Obtain []lineInfo with prefixParts and node text
			lines := formatHierarchy([]*calculationNode{rootNode})

			// Clear previous contents
			hierarchyContainer.Objects = nil

			// Render each line as an HBox of colored canvas.Text segments
			for _, ln := range lines {
				segments := []fyne.CanvasObject{}
				depth := len(ln.PrefixParts) - 1 // actual depth in tree
				// Build prefixes for each ancestor depth
				for lvl := 0; lvl < depth; lvl++ {
					if ln.PrefixParts[lvl] {
						// ancestor was last → spaces
						txt := canvas.NewText("    ", color.White) // white for blank
						txt.TextStyle = fyne.TextStyle{Monospace: true}
						segments = append(segments, txt)
					} else {
						// draw vertical line in color=palette[lvl]
						txt := canvas.NewText("│   ", palette[lvl%len(palette)])
						txt.TextStyle = fyne.TextStyle{Monospace: true}
						segments = append(segments, txt)
					}
				}
				// Now draw branch symbol ("├── " or "└── ") in color=palette[depth]
				branchSymbol := "├── "
				if ln.IsLast {
					branchSymbol = "└── "
				}
				brText := canvas.NewText(branchSymbol, palette[depth%len(palette)])
				brText.TextStyle = fyne.TextStyle{Monospace: true}
				segments = append(segments, brText)

				// Finally the node text, also in same color
				nodeTxt := canvas.NewText(ln.Text, palette[depth%len(palette)])
				nodeTxt.TextStyle = fyne.TextStyle{Monospace: true}
				segments = append(segments, nodeTxt)

				// Wrap segments in an HBox and add to the container
				hierarchyContainer.Add(container.NewHBox(segments...))
			}
			hierarchyContainer.Refresh()
		}

		statusLabel.SetText(fmt.Sprintf("Calculation result for %s %.2f %s:",
			data.GetAlloyNameByID(selected), amt, mode,
		))

		// Build summaryData
		summaryData = [][]string{{"Material", "mB", "Ingots"}}
		var ids []string
		for id := range finalMB {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool {
			return data.GetAlloyNameByID(ids[i]) < data.GetAlloyNameByID(ids[j])
		})
		for _, id := range ids {
			mbVal := finalMB[id]
			summaryData = append(summaryData, []string{
				data.GetAlloyNameByID(id),
				fmt.Sprintf("%.2f", mbVal),
				fmt.Sprintf("%.3f", mbVal/100.0),
			})
		}
		summaryTable.Refresh()
	})

	// Build left panel: controls + accordion
	inputForm := container.NewVBox(
		widget.NewLabel("Target Alloy:"),
		alloySelector,
		widget.NewLabel("Amount:"),
		amountEntry,
		widget.NewLabel("Mode:"),
		modeRadio,
	)
	leftPanel := container.NewBorder(inputForm, calcButton, nil, nil, container.NewVScroll(percentageAccordion))

	// Build right panel: status + hierarchy + summary
	statusLabel = widget.NewLabel("Enter data and press Calculate.")
	statusLabel.Wrapping = fyne.TextWrapWord

	hierarchyLabel := widget.NewLabelWithStyle("Calculation Hierarchy:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	summaryLabel := widget.NewLabelWithStyle("Final Summary (Base Materials):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	hierarchySection := container.NewBorder(hierarchyLabel, nil, nil, nil, container.NewScroll(hierarchyContainer))
	summarySection := container.NewBorder(summaryLabel, nil, nil, nil, container.NewVScroll(summaryTable))
	rightSplit := container.NewVSplit(hierarchySection, summarySection)
	rightSplit.SetOffset(0.6)

	// Top area (status), and below it the resizable split
	rightContent := container.NewBorder(
		statusLabel,
		nil,
		nil,
		nil,
		rightSplit,
	)

	// Main split: leftPanel vs rightContent (draggable divider)
	mainSplit := container.NewHSplit(leftPanel, rightContent)
	mainSplit.SetOffset(0.35)

	win.SetContent(mainSplit)
	win.SetPadded(true)
	win.Resize(fyne.NewSize(1100, 700))

	return win
}
