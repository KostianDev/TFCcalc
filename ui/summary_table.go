package ui

import (
	"fmt"
	"sort"
	"tfccalc/data"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Summary table helpers.

// InitSummaryTable constructs a table with columns: Material | mB | Ingots.
func InitSummaryTable() *widget.Table {
	summaryData = [][]string{{"Material", "mB", "Ingots"}}

	table := widget.NewTable(
		// Number of rows and columns.
		func() (int, int) {
			return len(summaryData), 3
		},
		// Create a new cell for each entry.
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.Alignment = fyne.TextAlignLeading
			return container.NewPadded(lbl)
		},
		// Update a cell based on row/col.
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			cont := cell.(*fyne.Container)
			lbl := cont.Objects[0].(*widget.Label)
			if id.Row < len(summaryData) && id.Col < len(summaryData[id.Row]) {
				lbl.SetText(summaryData[id.Row][id.Col])
				if id.Row == 0 {
					// Header row: bold and centered.
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
			lbl.Refresh()
		},
	)
	table.SetColumnWidth(0, 200)
	table.SetColumnWidth(1, 100)
	table.SetColumnWidth(2, 100)
	return table
}

// UpdateSummaryData rebuilds summaryData from finalMB and refreshes the table.
func UpdateSummaryData(finalMB map[string]float64, table *widget.Table) {
	// Start with just the header.
	summaryData = [][]string{{"Material", "mB", "Ingots"}}

	// Sort keys by alloy name.
	var ids []string
	for id := range finalMB {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return data.GetAlloyNameByID(ids[i]) < data.GetAlloyNameByID(ids[j])
	})

	// Append each alloy row in sorted order.
	for _, id := range ids {
		mbVal := finalMB[id]
		summaryData = append(summaryData, []string{
			data.GetAlloyNameByID(id),
			fmt.Sprintf("%.2f", mbVal),
			fmt.Sprintf("%.3f", mbVal/100.0),
		})
	}
	table.Refresh()
}
