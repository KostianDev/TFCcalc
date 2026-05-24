package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var (
	alloyNames []string
	alloyIDs   map[string]string

	// Slider controls per alloy ingredient.
	alloyPercentageControls map[string]map[string]*percentageControl

	// Warning label per alloy for auto-clamp notices.
	alloyPercentageWarnings map[string]*widget.Label

	// Guard per alloy to avoid recursive slider updates.
	alloyPercentageUpdating map[string]bool

	// Accordion that holds all "Configure: <Alloy>" sections.
	percentageAccordion *widget.Accordion

	// VBox container for colored ASCII tree lines.
	hierarchyContainer *fyne.Container

	// Summary table (Material, mB, Ingots).
	summaryTable *widget.Table
	// Summary table data rows.
	summaryData [][]string

	// Currently selected alloy ID.
	currentAlloyID string

	// Amount input field.
	amountEntry *widget.Entry

	// Mode selector (mB or Ingots).
	modeRadio *widget.RadioGroup

	// Status message label.
	statusLabel *widget.Label
)
