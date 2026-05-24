package adapterui

import (
	"fmt"
	"strconv"

	"tfccalc/domain"
	forginguc "tfccalc/usecase/forging"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// BuildAnvilTab constructs the Anvil tab UI and wires it to the forging service.
func BuildAnvilTab(svc *forginguc.Service) fyne.CanvasObject {
	// Target input (integer 0..150)
	targetEntry := widget.NewEntry()
	targetEntry.PlaceHolder = "Target value (0-150)"

	// Helper to create a select for final-slot: Any / Hit / Stamp / Bend / Upset / Shrink / Draw
	options := []string{"Any", "Hit", "Stamp", "Bend", "Upset", "Shrink", "Draw"}
	selA := widget.NewSelect(options, nil)
	selB := widget.NewSelect(options, nil)
	selC := widget.NewSelect(options, nil)
	selA.PlaceHolder = "Any"
	selB.PlaceHolder = "Any"
	selC.PlaceHolder = "Any"

	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapWord

	solveButton := widget.NewButton("Solve", func() {
		// Clear previous result
		resultLabel.SetText("")

		// Parse target
		tstr := targetEntry.Text
		tval, err := strconv.Atoi(tstr)
		if err != nil || tval < 0 || tval > 150 {
			resultLabel.SetText("Enter a valid integer target between 0 and 150.")
			return
		}

		// Build pattern
		var pattern [3]domain.FinalRequirement
		sels := []string{selA.Selected, selB.Selected, selC.Selected}
		for i, s := range sels {
			switch s {
			case "", "Any":
				pattern[i] = domain.FinalRequirement{Kind: domain.FinalAny}
			case "Hit":
				pattern[i] = domain.FinalRequirement{Kind: domain.FinalHit}
			default:
				// Exact action
				var act domain.ForgeAction
				switch s {
				case "Stamp":
					act = domain.Stamp
				case "Bend":
					act = domain.Bend
				case "Upset":
					act = domain.Upset
				case "Shrink":
					act = domain.Shrink
				case "Draw":
					act = domain.Draw
				default:
					pattern[i] = domain.FinalRequirement{Kind: domain.FinalAny}
					continue
				}
				pattern[i] = domain.FinalRequirement{Kind: domain.FinalExact, Action: act}
			}
		}

		// Solve
		seq, err := svc.SolveForTarget(tval, pattern)
		if err != nil {
			resultLabel.SetText(fmt.Sprintf("No solution: %v", err))
			return
		}

		// Format sequence for display
		var out string
		for i, a := range seq {
			name := humanActionName(a)
			// mark last three
			if i >= len(seq)-3 {
				out += fmt.Sprintf("[%s] ", name)
			} else {
				out += fmt.Sprintf("%s ", name)
			}
		}
		resultLabel.SetText(out)
	})

	// Layout
	form := container.NewVBox(
		widget.NewLabel("Anvil Forging"),
		widget.NewLabel("Target value (0..150):"),
		targetEntry,
		widget.NewLabel("Final three actions (in order, leave empty for Any):"),
		container.NewGridWithColumns(3, selA, selB, selC),
		solveButton,
		widget.NewLabel("Result sequence (last three shown in brackets):"),
		resultLabel,
	)

	// Initialize selects to Any
	selA.SetSelected("Any")
	selB.SetSelected("Any")
	selC.SetSelected("Any")

	return form
}

func humanActionName(a domain.ForgeAction) string {
	switch a {
	case domain.WeakHit:
		return "Weak Hit"
	case domain.MediumHit:
		return "Medium Hit"
	case domain.StrongHit:
		return "Strong Hit"
	case domain.Draw:
		return "Draw"
	case domain.Stamp:
		return "Stamp"
	case domain.Bend:
		return "Bend"
	case domain.Upset:
		return "Upset"
	case domain.Shrink:
		return "Shrink"
	default:
		return string(a)
	}
}
