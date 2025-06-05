package ui

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"tfccalc/calculator"
	"tfccalc/data"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var (
	alloyNames             []string                                    // Slice to store the names of available alloys.
	alloyIDs               = map[string]string{}                       // Map to store the ID of each alloy, using its name as the key.
	alloyPercentageEntries = make(map[string]map[string]*widget.Entry) // Map to store the user input fields for alloy percentages. The outer key is the alloy ID, and the inner map uses ingredient IDs as keys to access the corresponding entry field.
	percentageAccordion    *widget.Accordion                           // Accordion widget to display and manage user-adjustable alloy percentages.
	resultTree             *widget.Tree                                // Tree widget to display the hierarchical breakdown of alloy ingredients.
	treeRoots              []*calculationNode                          // Slice to store the root nodes of the calculation tree.
	treeNodes              map[widget.TreeNodeID]*calculationNode      // Map to quickly access any node in the tree using its ID.
	summaryTable           *widget.Table                               // Table widget to display a summary of the required base materials.
	summaryData            [][]string                                  // Two-dimensional slice to hold the data for the summary table.
	currentAlloyID         string                                      // Stores the ID of the currently selected alloy.
	amountEntry            *widget.Entry                               // Input field for the desired amount of the target alloy.
	modeRadio              *widget.RadioGroup                          // Radio group to select the calculation mode (by mB or by Ingots).
	statusLabel            *widget.Label                               // Label to display status messages and calculation results.
)

// calculationNode represents a node in the ingredient breakdown tree.
type calculationNode struct {
	ID           string             // Unique identifier for the node.
	AlloyID      string             // ID of the alloy or material this node represents.
	Name         string             // Display name of the alloy or material.
	AmountMB     float64            // Amount in milliBuckets (mB).
	AmountIngots float64            // Amount in Ingots.
	IsSummary    bool               // Indicates if this node is part of the summary.
	IsBaseMetal  bool               // Indicates if this node represents a base metal.
	IsSeparator  bool               // Indicates if this node is a separator (for visual purposes).
	Children     []*calculationNode // Slice of child nodes in the tree.
}

// updateTreeData updates the data source for the result tree and rebuilds the node map.
func updateTreeData(newRoots []*calculationNode) {
	treeRoots = newRoots
	treeNodes = make(map[widget.TreeNodeID]*calculationNode)
	var walk func(*calculationNode)
	walk = func(node *calculationNode) {
		if node == nil {
			return
		}
		treeNodes[node.ID] = node
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, root := range newRoots {
		walk(root)
	}
}

// treeChildren returns the IDs of the child nodes for a given node ID in the tree.
// If the ID is empty, it returns the IDs of the root nodes.
func treeChildren(id widget.TreeNodeID) []widget.TreeNodeID {
	if id == "" {
		ids := make([]widget.TreeNodeID, len(treeRoots))
		for i, root := range treeRoots {
			ids[i] = root.ID
		}
		return ids
	}
	node, ok := treeNodes[id]
	if !ok || len(node.Children) == 0 {
		return []widget.TreeNodeID{}
	}
	ids := make([]widget.TreeNodeID, len(node.Children))
	for i, child := range node.Children {
		ids[i] = child.ID
	}
	return ids
}

// treeIsBranch returns true if a given node ID represents a branch (has children) in the tree.
func treeIsBranch(id widget.TreeNodeID) bool {
	if id == "" {
		return true
	}
	node, ok := treeNodes[id]
	return ok && len(node.Children) > 0
}

// treeCreateNode creates a new canvas object to represent a node in the tree.
// It consists of labels for the material name, amount in mB, and amount in Ingots.
func treeCreateNode(isBranch bool) fyne.CanvasObject {
	nameLabel := widget.NewLabel("Material")
	mbLabel := widget.NewLabel("0.00")
	mbLabel.Alignment = fyne.TextAlignTrailing
	ingotLabel := widget.NewLabel("0.000")
	ingotLabel.Alignment = fyne.TextAlignTrailing
	rightBox := container.NewHBox(mbLabel, widget.NewLabel("|"), ingotLabel)
	hbox := container.NewHBox(nameLabel, layout.NewSpacer(), rightBox)
	return hbox
}

// treeUpdateNode updates the content of a node widget in the tree with the data from the corresponding calculationNode.
// It sets the text of the labels to display the material name and amounts.
func treeUpdateNode(id widget.TreeNodeID, isBranch bool, nodeWidget fyne.CanvasObject) {
	nodeData, ok := treeNodes[id]
	if !ok {
		log.Printf("!!! Node not found in treeNodes for ID: %s", id)
		if hbox, okW := nodeWidget.(*fyne.Container); okW && len(hbox.Objects) > 0 {
			if nameLabel, okL := hbox.Objects[0].(*widget.Label); okL {
				nameLabel.SetText("Error: node " + string(id) + "?")
			}
		}
		return
	}
	hbox, okH := nodeWidget.(*fyne.Container)
	if !okH || len(hbox.Objects) < 3 {
		log.Printf("Error: invalid type or structure of the node widget (HBox)")
		return
	}
	nameLabel, okN := hbox.Objects[0].(*widget.Label)
	rightBox, okR := hbox.Objects[2].(*fyne.Container)
	if !okN || !okR || len(rightBox.Objects) < 3 {
		log.Printf("Error: invalid structure of the right part of the node widget")
		return
	}
	mbLabel, okMB := rightBox.Objects[0].(*widget.Label)
	ingotLabel, okI := rightBox.Objects[2].(*widget.Label)
	if !okMB || !okI {
		log.Printf("Error: invalid types in the right part of the node widget")
		return
	}
	nameLabel.SetText(nodeData.Name)
	rightBox.Show()
	if nodeData.IsSeparator {
		nameLabel.Alignment = fyne.TextAlignCenter
		nameLabel.TextStyle.Bold = true
		rightBox.Hide()
	} else {
		nameLabel.Alignment = fyne.TextAlignLeading
		nameLabel.TextStyle.Bold = isBranch && !nodeData.IsSummary
		if nodeData.IsSummary && !nodeData.IsBaseMetal {
			rightBox.Hide()
		} else if nodeData.AmountMB > 0 || nodeData.IsBaseMetal {
			mbLabel.SetText(fmt.Sprintf("%.2f", nodeData.AmountMB))
			ingotLabel.SetText(fmt.Sprintf("%.3f", nodeData.AmountIngots))
		} else {
			rightBox.Hide()
		}
	}
	nameLabel.Refresh()
}

// buildResultTreeRecursive recursively builds the ingredient breakdown tree for a given alloy and amount.
// It takes the alloy ID, amount in mB, user-defined percentages, a map to track visited alloys to prevent cycles,
// the current recursion level, and the maximum recursion level as input.
func buildResultTreeRecursive(alloyID string, amountMB float64, percentages map[string]map[string]float64, visited map[string]int, level int, maxLevel int) (*calculationNode, error) {
	if level > maxLevel {
		return nil, nil
	}
	nodeUID := fmt.Sprintf("%s_lvl%d_%d", alloyID, level, visited[alloyID])
	visited[alloyID]++
	alloyData, ok := data.GetAlloyByID(alloyID)
	if !ok {
		return nil, fmt.Errorf("unknown material in tree: %s", alloyID)
	}
	node := &calculationNode{
		ID:           nodeUID,
		AlloyID:      alloyID,
		Name:         alloyData.Name,
		AmountMB:     amountMB,
		AmountIngots: amountMB / 100.0,
		Children:     []*calculationNode{},
		IsBaseMetal:  alloyData.Type == "base",
	}
	idForIngredients := alloyID
	recipeSourceAlloy := alloyData
	processedChildren := false
	if alloyData.Type == "final_steel" {
		// Use RawForm for ingredient breakdown first
		if alloyData.RawFormID.Valid {
			idForIngredients = alloyData.RawFormID.String
			recipeSourceAlloy, ok = data.GetAlloyByID(idForIngredients)
			if !ok {
				return nil, fmt.Errorf("raw_form %s not found for %s", idForIngredients, alloyID)
			}
		}
		node.Name = fmt.Sprintf("%s (%.2fmB)", alloyData.Name, amountMB)
		// Recurse RawForm
		if alloyData.RawFormID.Valid {
			rawNode, err := buildResultTreeRecursive(idForIngredients, amountMB, percentages, visited, level+1, maxLevel)
			if err != nil {
				return nil, err
			}
			if rawNode != nil {
				node.Children = append(node.Children, rawNode)
			}
		}
		// Recurse ExtraIngredient
		if alloyData.ExtraIngredientID.Valid {
			extraNode, err := buildResultTreeRecursive(alloyData.ExtraIngredientID.String, amountMB, percentages, visited, level+1, maxLevel)
			if err != nil {
				return nil, err
			}
			if extraNode != nil {
				node.Children = append(node.Children, extraNode)
			}
		}
		processedChildren = true
	} else if alloyData.Type == "processed" && alloyID == "steel" {
		// Steel is 100% pig_iron
		node.Name = fmt.Sprintf("%s (%.2fmB)", alloyData.Name, amountMB)
		pigIronNode, err := buildResultTreeRecursive("pig_iron", amountMB, percentages, visited, level+1, maxLevel)
		if err != nil {
			return nil, err
		}
		if pigIronNode != nil {
			node.Children = append(node.Children, pigIronNode)
		}
		processedChildren = true
	}
	if !processedChildren && alloyData.Type != "base" && len(recipeSourceAlloy.Ingredients) > 0 {
		// Standard alloy/raw_steel breakdown
		node.Name = fmt.Sprintf("%s (%.2fmB)", recipeSourceAlloy.Name, amountMB)
		currentPercentages, percErr := calculator.GetDefaultPercentages(idForIngredients)
		if percErr == nil {
			if specPerc, found := percentages[idForIngredients]; found {
				fullPercMap := make(map[string]float64)
				for k, v := range specPerc {
					fullPercMap[k] = v
				}
				// Fill missing with defaults
				for _, ing := range recipeSourceAlloy.Ingredients {
					if _, exists := fullPercMap[ing.IngredientID]; !exists {
						if defPercVal, defExists := currentPercentages[ing.IngredientID]; defExists {
							fullPercMap[ing.IngredientID] = defPercVal
						}
					}
				}
				if valid, _ := calculator.ValidatePercentages(idForIngredients, fullPercMap); valid {
					currentPercentages = fullPercMap
				}
			}
		} else {
			return nil, fmt.Errorf("error getting %% for %s in tree: %w", idForIngredients, percErr)
		}
		if validFin, finErr := calculator.ValidatePercentages(idForIngredients, currentPercentages); !validFin {
			return nil, fmt.Errorf("invalid final %% for %s in tree: %w", idForIngredients, finErr)
		}
		for _, ing := range recipeSourceAlloy.Ingredients {
			percentage := currentPercentages[ing.IngredientID]
			childAmountMB := amountMB * (percentage / 100.0)
			if childAmountMB < 0.001 {
				continue
			}
			childNode, err := buildResultTreeRecursive(ing.IngredientID, childAmountMB, percentages, visited, level+1, maxLevel)
			if err != nil {
				log.Printf("Error building branch %s for %s: %v", ing.IngredientID, alloyID, err)
				continue
			}
			if childNode != nil {
				node.Children = append(node.Children, childNode)
			}
		}
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].Name < node.Children[j].Name
		})
	} else if !processedChildren {
		node.Name = alloyData.Name
	}
	return node, nil
}

// createPercentageInputsForAlloy creates the UI elements (labels and entry fields) for adjusting the ingredient percentages of a given alloy.
func createPercentageInputsForAlloy(alloyID string) (fyne.CanvasObject, error) {
	alloy, ok := data.GetAlloyByID(alloyID)
	if !ok {
		return nil, fmt.Errorf("alloy %s not found", alloyID)
	}
	if len(alloy.Ingredients) == 0 {
		return widget.NewLabel("  (Percentages are not configurable)"), nil
	}
	content := container.NewVBox()
	currentAlloyEntries := make(map[string]*widget.Entry)
	alloyPercentageEntries[alloyID] = currentAlloyEntries
	defaultPercentages, _ := calculator.GetDefaultPercentages(alloyID)
	for _, ing := range alloy.Ingredients {
		ingName := data.GetAlloyNameByID(ing.IngredientID)
		label := widget.NewLabel(fmt.Sprintf("%s [%.0f-%.0f%%]:", ingName, ing.Min, ing.Max))
		entry := widget.NewEntry()
		entry.Validator = validation.NewRegexp(`^\d+(\.\d+)?$`, "Number")
		if defaultPercentages != nil {
			if defPerc, found := defaultPercentages[ing.IngredientID]; found {
				entry.PlaceHolder = fmt.Sprintf("%.1f", defPerc)
			} else {
				entry.PlaceHolder = "???"
			}
		}
		entry.Wrapping = fyne.TextTruncate
		currentAlloyEntries[ing.IngredientID] = entry
		content.Add(container.NewGridWithColumns(2, label, entry))
	}
	return content, nil
}

// buildAccordionItemsRecursive recursively builds the accordion items for adjusting alloy percentages.
// It traverses the alloy ingredient tree and creates an accordion item for each alloy that has configurable percentages.
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
	alloyForInputs := alloy
	if alloy.Type == "final_steel" {
		idForInputs = alloy.RawFormID.String
		alloyForInputs, ok = data.GetAlloyByID(idForInputs)
		if !ok {
			return
		}
	}
	if len(alloyForInputs.Ingredients) > 0 {
		content, err := createPercentageInputsForAlloy(idForInputs)
		if err != nil {
			content = widget.NewLabel(fmt.Sprintf("Error loading fields: %v", err))
		}
		accordionItem := widget.NewAccordionItem(fmt.Sprintf("Configure: %s", alloyForInputs.Name), content)
		acc.Append(accordionItem)
		for _, ing := range alloyForInputs.Ingredients {
			ingAlloy, ingOk := data.GetAlloyByID(ing.IngredientID)
			if !ingOk {
				continue
			}
			nextID := ing.IngredientID
			if ingAlloy.Type == "final_steel" {
				nextID = ingAlloy.RawFormID.String
			}
			nextAlloy, nextOk := data.GetAlloyByID(nextID)
			if nextOk && (nextAlloy.Type == "alloy" || nextAlloy.Type == "raw_steel") && len(nextAlloy.Ingredients) > 0 {
				buildAccordionItemsRecursive(nextID, acc, visited)
			}
		}
	} else if alloyForInputs.Type == "alloy" || alloyForInputs.Type == "raw_steel" {
		acc.Append(widget.NewAccordionItem(fmt.Sprintf("Configure: %s", alloyForInputs.Name), widget.NewLabel(" (No configurable ingredients)")))
	}
}

// BuildUI creates and returns the main window of the application.
// It initializes all UI elements, sets up event handlers, and arranges the layout.
func BuildUI(app fyne.App) fyne.Window {
	resouceIcon, err := fyne.LoadResourceFromPath("./assets/tfc_icon.png")
	if err != nil {
		log.Println("Error loading resource icon:", err)
	}

	win := app.NewWindow("TFC Alloy Calculator")
	win.SetIcon(resouceIcon)
	win.SetMaster()

	// Build the alloy selector from the database
	alloyNames = []string{}
	alloyIDs = make(map[string]string)
	alloys := data.GetAllAlloys()
	for id, alloyData := range alloys {
		if alloyData.Type == "alloy" || alloyData.Type == "final_steel" {
			alloyNames = append(alloyNames, alloyData.Name)
			alloyIDs[alloyData.Name] = id
		}
	}
	sort.Strings(alloyNames)

	alloySelector := widget.NewSelect(alloyNames, func(selectedName string) {
		newID := alloyIDs[selectedName]
		if currentAlloyID == newID {
			return
		}
		currentAlloyID = newID
		log.Println("Selected alloy:", selectedName, "(ID:", currentAlloyID, ")")
		alloyPercentageEntries = make(map[string]map[string]*widget.Entry)
		if percentageAccordion == nil {
			log.Println("Accordion is nil!")
			return
		}
		percentageAccordion.Items = []*widget.AccordionItem{}
		visited := make(map[string]bool)
		startID := currentAlloyID
		alloyData, _ := data.GetAlloyByID(currentAlloyID)
		if alloyData.Type == "final_steel" {
			startID = alloyData.RawFormID.String
		}
		buildAccordionItemsRecursive(startID, percentageAccordion, visited)
		percentageAccordion.Refresh()
		if len(percentageAccordion.Items) > 0 {
			percentageAccordion.Open(0)
		} else {
			noSettingsItem := widget.NewAccordionItem("Percentage Configuration", widget.NewLabel("No configurable ingredients for this alloy."))
			noSettingsItem.Open = true
			percentageAccordion.Append(noSettingsItem)
			percentageAccordion.Refresh()
		}
		if resultTree == nil {
			log.Println("resultTree is nil during alloy change!")
			return
		}
		updateTreeData([]*calculationNode{})
		resultTree.Refresh()
		summaryData = [][]string{}
		if summaryTable != nil {
			summaryTable.Refresh()
		}
		statusLabel.SetText("Select amount and mode, then press 'Calculate'.")
	})
	alloySelector.PlaceHolder = "Select alloy..."

	amountEntry = widget.NewEntry()
	amountEntry.SetPlaceHolder("Amount...")
	amountEntry.Validator = validation.NewRegexp(`^\d+(\.\d+)?$`, "Number > 0")

	modeRadio = widget.NewRadioGroup([]string{"mB", "Ingots"}, nil)
	modeRadio.Horizontal = true
	modeRadio.SetSelected("Ingots")

	percentageAccordion = widget.NewAccordion()

	statusLabel = widget.NewLabel("Enter data and press 'Calculate'.")
	statusLabel.Wrapping = fyne.TextWrapWord

	treeRoots = []*calculationNode{}
	treeNodes = make(map[widget.TreeNodeID]*calculationNode)
	resultTree = widget.NewTree(treeChildren, treeIsBranch, treeCreateNode, treeUpdateNode)
	resultTree.OnBranchClosed = func(uid widget.TreeNodeID) {}
	resultTree.OnBranchOpened = func(uid widget.TreeNodeID) {}

	summaryData = [][]string{}
	summaryTable = widget.NewTable(
		func() (int, int) {
			return len(summaryData), 3
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignLeading
			return container.NewPadded(label) // Use container for proper padding
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			cont := cell.(*fyne.Container)
			label := cont.Objects[0].(*widget.Label)
			if id.Row >= 0 && id.Row < len(summaryData) && id.Col >= 0 && id.Col < len(summaryData[id.Row]) {
				label.SetText(summaryData[id.Row][id.Col])
				// Style and alignment
				if id.Row == 0 {
					label.TextStyle.Bold = true
					label.Alignment = fyne.TextAlignCenter
				} else {
					label.TextStyle.Bold = false
					switch id.Col {
					case 0:
						label.Alignment = fyne.TextAlignLeading
					case 1, 2:
						label.Alignment = fyne.TextAlignTrailing
					}
				}
			} else {
				label.SetText("")
			}
			label.Refresh()
		},
	)
	summaryTable.SetColumnWidth(0, 200)
	summaryTable.SetColumnWidth(1, 100)
	summaryTable.SetColumnWidth(2, 100)

	calculateButton := widget.NewButton("Calculate", func() {
		statusLabel.SetText("Calculating...")
		selectedAlloyID := currentAlloyID
		if selectedAlloyID == "" {
			statusLabel.SetText("Error: Alloy not selected.")
			return
		}
		amountStr := amountEntry.Text
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			statusLabel.SetText("Error: Enter a valid positive amount.")
			return
		}
		mode := modeRadio.Selected
		if mode == "" {
			statusLabel.SetText("Error: Select a calculation mode (mB or Ingots).")
			return
		}
		allUserPercentages := make(map[string]map[string]float64)
		validationErrors := []string{}
		for alloyID, entriesMap := range alloyPercentageEntries {
			currentAlloyUserPercentages := make(map[string]float64)
			useCurrentCustom := false
			defaultPercentages, _ := calculator.GetDefaultPercentages(alloyID)
			alloyData, alloyExists := data.GetAlloyByID(alloyID)
			if !alloyExists {
				continue
			}
			for ingID, entry := range entriesMap {
				if entry.Text != "" {
					percent, err := strconv.ParseFloat(entry.Text, 64)
					if err != nil {
						validationErrors = append(validationErrors, fmt.Sprintf("Invalid %% for %s in %s", data.GetAlloyNameByID(ingID), data.GetAlloyNameByID(alloyID)))
						continue
					}
					currentAlloyUserPercentages[ingID] = percent
					useCurrentCustom = true
				}
			}
			if useCurrentCustom || len(alloyData.Ingredients) > 0 {
				finalPercMap := make(map[string]float64)
				for k, v := range currentAlloyUserPercentages {
					finalPercMap[k] = v
				}
				if defaultPercentages != nil {
					for _, ing := range alloyData.Ingredients {
						if _, exists := finalPercMap[ing.IngredientID]; !exists {
							if defPercVal, defExists := defaultPercentages[ing.IngredientID]; defExists {
								finalPercMap[ing.IngredientID] = defPercVal
							} else {
								validationErrors = append(validationErrors, fmt.Sprintf("No default for %s in %s", data.GetAlloyNameByID(ing.IngredientID), data.GetAlloyNameByID(alloyID)))
							}
						}
					}
				}
				valid, valErr := calculator.ValidatePercentages(alloyID, finalPercMap)
				if !valid {
					validationErrors = append(validationErrors, fmt.Sprintf("Error in %% for %s: %v", data.GetAlloyNameByID(alloyID), valErr))
				} else if len(finalPercMap) > 0 {
					allUserPercentages[alloyID] = finalPercMap
				}
			}
		}
		if len(validationErrors) > 0 {
			statusLabel.SetText("Percentage input errors:\n- " + strings.Join(validationErrors, "\n- "))
			return
		}
		var percentagesForCalc map[string]map[string]float64 = nil
		if len(allUserPercentages) > 0 {
			percentagesForCalc = allUserPercentages
		}
		finalBaseMB, _, calcErr := calculator.CalculateRequirements(selectedAlloyID, amount, mode, percentagesForCalc)
		if calcErr != nil {
			statusLabel.SetText(fmt.Sprintf("Calculation error:\n%v", calcErr))
			updateTreeData([]*calculationNode{})
			resultTree.Refresh()
			summaryData = [][]string{}
			summaryTable.Refresh()
		} else {
			statusLabel.SetText(fmt.Sprintf("Calculation result for %s %.2f %s:", data.GetAlloyNameByID(selectedAlloyID), amount, mode))
			rootAmountMB := amount
			if mode == "Ingots" {
				rootAmountMB = amount * 100.0
			}
			treeStartID := selectedAlloyID
			rootNode, treeErr := buildResultTreeRecursive(treeStartID, rootAmountMB, percentagesForCalc, make(map[string]int), 0, 5)
			if treeErr != nil {
				statusLabel.SetText(fmt.Sprintf("Error building tree: %v", treeErr))
				updateTreeData([]*calculationNode{})
			} else {
				updateTreeData([]*calculationNode{rootNode})
			}
			resultTree.Refresh()
			if rootNode != nil {
				resultTree.OpenAllBranches()
			}
			summaryData = [][]string{{"Material", "mB", "Ingots"}}
			sortedIDs := make([]string, 0, len(finalBaseMB))
			for id := range finalBaseMB {
				sortedIDs = append(sortedIDs, id)
			}
			sort.Slice(sortedIDs, func(i, j int) bool {
				return data.GetAlloyNameByID(sortedIDs[i]) < data.GetAlloyNameByID(sortedIDs[j])
			})
			for _, id := range sortedIDs {
				mbVal := finalBaseMB[id]
				row := []string{data.GetAlloyNameByID(id), fmt.Sprintf("%.2f", mbVal), fmt.Sprintf("%.3f", mbVal/100.0)}
				summaryData = append(summaryData, row)
			}
			log.Printf("Data for summary table (summaryData): %v", summaryData)
			summaryTable.Refresh()
		}
	})

	inputForm := container.NewVBox(
		widget.NewLabel("Target Alloy:"),
		alloySelector,
		widget.NewLabel("Amount:"),
		amountEntry,
		widget.NewLabel("Mode:"),
		modeRadio,
	)
	percentageScroll := container.NewVScroll(percentageAccordion)
	percentageScroll.SetMinSize(fyne.NewSize(0, 180))
	leftPanel := container.NewBorder(inputForm, calculateButton, nil, nil, percentageScroll)

	treeLabel := widget.NewLabelWithStyle("Calculation Hierarchy:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	treeScroll := container.NewVScroll(resultTree)

	summaryLabel := widget.NewLabelWithStyle("Final Summary (Base Materials):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	summaryScroll := container.NewVScroll(summaryTable)
	summaryScroll.Content = summaryTable
	summaryContainer := container.NewBorder(
		summaryLabel,
		nil,
		nil,
		nil,
		container.NewStack( // Use Stack to fill the space
			summaryScroll,
			layout.NewSpacer(), // Add spacer
		),
	)

	resultsSplit := container.NewVSplit(
		container.NewBorder(treeLabel, nil, nil, nil, treeScroll),
		summaryContainer,
	)
	resultsSplit.Offset = 0.65

	rightPanel := container.NewBorder(
		statusLabel,
		nil,
		nil, nil,
		resultsSplit,
	)

	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.40

	win.SetContent(split)
	win.Resize(fyne.NewSize(1000, 700))
	win.SetFixedSize(false)

	// Initial placeholder in the accordion
	percentageAccordion.Append(widget.NewAccordionItem("Percentage Configuration", widget.NewLabel("Select an alloy to configure.")))

	return win
}
