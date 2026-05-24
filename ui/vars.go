package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var (
	alloyNames []string
	alloyIDs   map[string]string

	// Stores slider controls for each alloy ingredient percentage.
	alloyPercentageControls map[string]map[string]*percentageControl

	// Per-alloy warning label for auto-clamp messages.
	alloyPercentageWarnings map[string]*widget.Label

	// Per-alloy guard to avoid recursive slider updates.
	alloyPercentageUpdating map[string]bool

	// Accordion, куди ми кладемо всі “Configure: <Alloy>” пункти
	percentageAccordion *widget.Accordion

	// VBox-контейнер, у якому будуть кольорові рядки деревовидного ASCII
	hierarchyContainer *fyne.Container

	// Таблиця підсумкових матеріалів (Material, mB, Ingots)
	summaryTable *widget.Table
	// Дані для цієї таблиці (рядки)
	summaryData [][]string

	// ID поточного вибраного сплаву (заповнюється після Select)
	currentAlloyID string

	// Поле вводу бажаної кількості (Entry)
	amountEntry *widget.Entry

	// RadioGroup для вибору “mB” чи “Ingots”
	modeRadio *widget.RadioGroup

	// Label для статусних повідомлень
	statusLabel *widget.Label
)
