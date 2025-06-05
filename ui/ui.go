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

//
// This file ties everything together:
//  1) Alloy selector (Select dropdown)
//  2) Amount entry (Entry) + Mode radio (RadioGroup)
//  3) Percentage accordion
//  4) Tree rendering (calls formatHierarchy → RenderLines)
//  5) Summary table updates
//
// BuildUI(app) constructs a fx.Window, lays out controls on the left,
// and puts status + hierarchy + summary on the right. The “Calculate”
// callback triggers buildResultTreeRecursive → formatHierarchy → RenderLines,
// then calls UpdateSummaryData() for the summary.
//
// Global state (alloyNames, alloyIDs, percentage entries, etc.) all come from vars.go.
//

// BuildUI creates and returns the main window of the application.
func BuildUI(app fyne.App) fyne.Window {
	// 0) Predefined color palette: must match the one in tree_renderer.go
	palette := []color.Color{
		color.RGBA{R: 255, G: 102, B: 102, A: 255}, // Light Red
		color.RGBA{R: 102, G: 255, B: 102, A: 255}, // Light Green
		color.RGBA{R: 102, G: 178, B: 255, A: 255}, // Light Blue
		color.RGBA{R: 255, G: 255, B: 102, A: 255}, // Light Yellow
		color.RGBA{R: 255, G: 153, B: 255, A: 255}, // Light Pink
		color.RGBA{R: 153, G: 255, B: 255, A: 255}, // Light Cyan
	}

	// 1) Load icon if available
	resIcon, err := fyne.LoadResourceFromPath("./assets/tfc_icon.png")
	if err != nil {
		log.Println("Error loading icon:", err)
	}

	win := app.NewWindow("TFC Alloy Calculator")
	win.SetIcon(resIcon)
	win.SetMaster()

	// 2) Initialize alloyNames + alloyIDs for the Select dropdown
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

		// When user chooses a new alloy, clear previous percentage fields and the tree.
		alloyPercentageEntries = make(map[string]map[string]*widget.Entry)
		percentageAccordion.Items = nil

		// Build accordion items recursively starting from the raw form if this is final_steel.
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
			noItem := widget.NewAccordionItem("Percentage Configuration",
				widget.NewLabel("No configurable ingredients for this alloy."))
			noItem.Open = true
			percentageAccordion.Append(noItem)
			percentageAccordion.Refresh()
		}

		// Clear tree and summary
		hierarchyContainer.Objects = nil
		hierarchyContainer.Refresh()

		summaryData = [][]string{{"Material", "mB", "Ingots"}}
		summaryTable.Refresh()

		statusLabel.SetText("Select amount and mode, then press Calculate.")
	})
	alloySelector.PlaceHolder = "Select alloy..."

	// 3) Amount entry
	amountEntry = widget.NewEntry()
	amountEntry.PlaceHolder = "Amount..."
	amountEntry.Validator = validation.NewRegexp(`^\d+(\.\d+)?$`, "Number > 0")

	// 4) Mode radio group (“mB” or “Ingots”)
	modeRadio = widget.NewRadioGroup([]string{"mB", "Ingots"}, nil)
	modeRadio.Horizontal = true
	modeRadio.SetSelected("Ingots")

	// 5) Status label (wrapped text)
	statusLabel = widget.NewLabel("Enter data and press Calculate.")
	statusLabel.Wrapping = fyne.TextWrapWord

	// 6) Percentage accordion inside a scroll container
	percentageAccordion = widget.NewAccordion()
	accordionScroll := container.NewVScroll(percentageAccordion)
	accordionScroll.SetMinSize(fyne.NewSize(0, 200))

	// 7) Hierarchy container (VBox) + scroll
	hierarchyContainer = container.NewVBox()
	hierarchyScroll := container.NewScroll(hierarchyContainer)
	hierarchyScroll.SetMinSize(fyne.NewSize(0, 300))

	// 8) Summary table setup
	summaryTable = InitSummaryTable()

	// 9) Calculate button: gathers input, builds tree, renders lines, updates summary.
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

		// 9.1) Collect user‐entered percentages into userPercs
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

		// 9.2) Build the calculation tree
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
			lines := formatHierarchy([]*calculationNode{rootNode})

			hierarchyContainer.Objects = nil
			for _, ln := range lines {
				var segments []fyne.CanvasObject
				depth := len(ln.PrefixParts) - 1
				// Draw ancestor bars/spaces
				for lvl := 0; lvl < depth; lvl++ {
					if ln.PrefixParts[lvl] {
						txt := canvas.NewText("    ", color.White)
						txt.TextStyle = fyne.TextStyle{Monospace: true}
						segments = append(segments, txt)
					} else {
						txt := canvas.NewText("│   ", palette[lvl%len(palette)])
						txt.TextStyle = fyne.TextStyle{Monospace: true}
						segments = append(segments, txt)
					}
				}
				// Draw branch symbol
				branchSymbol := "├── "
				if ln.IsLast {
					branchSymbol = "└── "
				}
				brText := canvas.NewText(branchSymbol, palette[depth%len(palette)])
				brText.TextStyle = fyne.TextStyle{Monospace: true}
				segments = append(segments, brText)
				// Draw node text
				nodeTxt := canvas.NewText(ln.Text, palette[depth%len(palette)])
				nodeTxt.TextStyle = fyne.TextStyle{Monospace: true}
				segments = append(segments, nodeTxt)

				hierarchyContainer.Add(container.NewHBox(segments...))
			}
			hierarchyContainer.Refresh()
		}

		statusLabel.SetText(fmt.Sprintf("Calculation result for %s %.2f %s:",
			data.GetAlloyNameByID(selected), amt, mode,
		))

		// 9.3) Update summary table
		UpdateSummaryData(finalMB, summaryTable)
	})

	// 10) Left panel: Select dropdown, Amount entry, Mode radio, Accordion, Button
	inputForm := container.NewVBox(
		widget.NewLabel("Target Alloy:"),
		alloySelector,
		widget.NewLabel("Amount:"),
		amountEntry,
		widget.NewLabel("Mode:"),
		modeRadio,
	)
	leftPanel := container.NewBorder(
		inputForm,
		calcButton,
		nil,
		nil,
		container.NewVScroll(percentageAccordion),
	)

	// 11) Right panel: Status label, then a VSplit of hierarchy + summary
	statusLabel = widget.NewLabel("Enter data and press Calculate.")
	statusLabel.Wrapping = fyne.TextWrapWord

	hierarchyLabel := widget.NewLabelWithStyle(
		"Calculation Hierarchy:",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	summaryLabel := widget.NewLabelWithStyle(
		"Final Summary (Base Materials):",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	hierarchySection := container.NewBorder(
		hierarchyLabel,
		nil,
		nil,
		nil,
		container.NewScroll(hierarchyContainer),
	)
	summarySection := container.NewBorder(
		summaryLabel,
		nil,
		nil,
		nil,
		container.NewVScroll(summaryTable),
	)
	rightSplit := container.NewVSplit(hierarchySection, summarySection)
	rightSplit.SetOffset(0.6)

	rightContent := container.NewBorder(
		statusLabel,
		nil,
		nil,
		nil,
		rightSplit,
	)

	// 12) Main HSplit: leftPanel | rightContent
	mainSplit := container.NewHSplit(leftPanel, rightContent)
	mainSplit.SetOffset(0.35)

	win.SetContent(mainSplit)
	win.SetPadded(true)
	win.Resize(fyne.NewSize(1100, 700))

	return win
}
