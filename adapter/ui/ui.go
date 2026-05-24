package adapterui

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"tfccalc/domain"
	uihelpers "tfccalc/ui/helpers"
	"tfccalc/usecase/alloy"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
)

// BuildUI creates and returns the main window of the application using Fyne.
func BuildUI(app fyne.App, svc *alloy.Service) fyne.Window {
	if svc == nil {
		log.Fatal("alloy service is required")
	}

	// Load icon if available.
	resIcon, err := fyne.LoadResourceFromPath("./assets/tfc_icon.png")
	if err != nil {
		log.Println("Error loading icon:", err)
	}

	win := app.NewWindow("TFC Alloy Calculator")
	win.SetIcon(resIcon)
	win.SetMaster()

	// Initialize alloyNames and alloyIDs for the dropdown.
	alloyNames = []string{}
	alloyIDs = make(map[string]string)
	for id, alloyData := range svc.GetAllAlloys() {
		if alloyData.Type == domain.AlloyTypeAlloy || alloyData.Type == domain.AlloyTypeFinalSteel {
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

		// Reset percentage controls and result panes on selection.
		alloyPercentageControls = make(map[string]map[string]*percentageControl)
		alloyPercentageWarnings = make(map[string]*widget.Label)
		alloyPercentageUpdating = make(map[string]bool)
		percentageAccordion.Items = nil

		// Build accordion items, starting from the raw form for final_steel.
		visited := make(map[string]bool)
		startID := currentAlloyID
		if alloyData, ok := svc.GetAlloyByID(currentAlloyID); ok && alloyData.Type == domain.AlloyTypeFinalSteel {
			if alloyData.RawFormID != nil {
				startID = *alloyData.RawFormID
			}
		}
		buildAccordionItemsRecursive(startID, percentageAccordion, visited, svc)
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

		// Clear tree and summary.
		hierarchyContainer.Objects = nil
		hierarchyContainer.Refresh()

		summaryData = [][]string{{"Material", "mB", "Ingots"}}
		summaryTable.Refresh()

		statusLabel.SetText("Select amount and mode, then press Calculate.")
	})
	alloySelector.PlaceHolder = "Select alloy..."

	// Amount entry.
	amountEntry = widget.NewEntry()
	amountEntry.PlaceHolder = "Amount..."
	amountEntry.Validator = validation.NewRegexp(`^\d+(\.\d+)?$`, "Number > 0")

	// Mode radio group (mB or Ingots).
	modeRadio = widget.NewRadioGroup([]string{"mB", "Ingots"}, nil)
	modeRadio.Horizontal = true
	modeRadio.SetSelected("Ingots")

	// Status label.
	statusLabel = widget.NewLabel("Enter data and press Calculate.")
	statusLabel.Wrapping = fyne.TextWrapWord

	// Percentage accordion inside a scroll container.
	percentageAccordion = widget.NewAccordion()
	alloyPercentageControls = make(map[string]map[string]*percentageControl)
	alloyPercentageWarnings = make(map[string]*widget.Label)
	alloyPercentageUpdating = make(map[string]bool)
	accordionScroll := container.NewVScroll(percentageAccordion)
	accordionScroll.SetMinSize(fyne.NewSize(0, 200))

	// Hierarchy container and scroll.
	hierarchyContainer = container.NewVBox()
	hierarchyScroll := container.NewScroll(hierarchyContainer)
	hierarchyScroll.SetMinSize(fyne.NewSize(0, 300))

	// Summary table setup.
	summaryTable = InitSummaryTable()

	// Calculate button: gather input, build tree, update summary.
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

		// Collect percentages from sliders.
		userPercs := make(map[string]map[string]float64)
		var validationErrors []string
		for alloyID, controlMap := range alloyPercentageControls {
			alloyInfo, _ := svc.GetAlloyByID(alloyID)
			finalPerc := make(map[string]float64)
			for ingID, ctl := range controlMap {
				finalPerc[ingID] = ctl.slider.Value
			}
			if len(alloyInfo.Ingredients) > 0 {
				valid, errv := svc.ValidatePercentages(alloyID, finalPerc)
				if !valid {
					validationErrors = append(
						validationErrors,
						fmt.Sprintf("Error in %% for %s: %v",
							svc.GetAlloyNameByID(alloyID),
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
		finalMB, _, errCalc := svc.CalculateRequirements(selected, amt, mode, percMap)
		if errCalc != nil {
			statusLabel.SetText(fmt.Sprintf("Calculation error:\n%v", errCalc))
			hierarchyContainer.Objects = nil
			hierarchyContainer.Refresh()
			summaryData = [][]string{{"Material", "mB", "Ingots"}}
			summaryTable.Refresh()
			return
		}

		// Build the calculation tree.
		rootMB := amt
		if mode == "Ingots" {
			rootMB = amt * 100.0
		}
		rootNode, errTree := uihelpers.BuildResultTreeRecursive(selected, rootMB, percMap, make(map[string]int), 0, 5, svc)
		if errTree != nil {
			statusLabel.SetText(fmt.Sprintf("Tree build error: %v", errTree))
			hierarchyContainer.Objects = nil
			hierarchyContainer.Refresh()
		} else if rootNode != nil {
			lines := uihelpers.FormatHierarchy([]*uihelpers.CalculationNode{rootNode})
			treeBox := RenderLines(lines)

			hierarchyContainer.Objects = nil
			hierarchyContainer.Objects = treeBox.Objects
			hierarchyContainer.Refresh()
		}

		statusLabel.SetText(fmt.Sprintf("Calculation result for %s %.2f %s:",
			svc.GetAlloyNameByID(selected), amt, mode,
		))

		// Update summary table.
		UpdateSummaryData(finalMB, summaryTable, svc)
	})

	// Left panel: inputs and controls.
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

	// Right panel: status, hierarchy, summary.
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

	// Main split: left and right panels.
	mainSplit := container.NewHSplit(leftPanel, rightContent)
	mainSplit.SetOffset(0.35)

	win.SetContent(mainSplit)
	win.SetPadded(true)
	win.Resize(fyne.NewSize(1100, 700))

	return win
}
