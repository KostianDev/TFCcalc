package adapterui

import (
	"fmt"
	"image/color"
	"strconv"

	"tfccalc/domain"
	forginguc "tfccalc/usecase/forging"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// BuildAnvilTab constructs the Anvil tab UI and wires it to the forging service.
func BuildAnvilTab(svc *forginguc.Service) fyne.CanvasObject {
	// Target input (integer 0..150)
	targetEntry := widget.NewEntry()
	targetEntry.PlaceHolder = "Target value (0-150)"

	// Helper options
	options := []string{"Any", "Hit", "Stamp", "Bend", "Upset", "Shrink", "Draw"}

	// makeIconDropdown builds an inline dropdown: a button row and a grid of icon buttons
	makeIconDropdown := func(initial string, onChange func(string)) (fyne.CanvasObject, func() string, func(string)) {
		cur := initial
		// preview icon + label button
		icon := canvas.NewImageFromResource(nil)
		icon.SetMinSize(fyne.NewSize(24, 24))
		icon.FillMode = canvas.ImageFillContain
		lbl := widget.NewLabel(cur)
		// wrap icon in fixed-size box so it doesn't get horizontally squashed
		iconBg := canvas.NewRectangle(color.Transparent)
		iconBg.SetMinSize(fyne.NewSize(24, 24))
		iconBox := container.NewStack(iconBg, container.NewCenter(icon))
		// we'll use a container to show icon + label
		display := container.NewHBox(iconBox, lbl)

		// options grid
		optsGrid := container.NewGridWithColumns(4)
		optsGrid.Hide()
		// popup will be created after the container is available; declare here so option closures can hide it
		var popup *widget.PopUp

		// populate
		for _, opt := range options {
			// determine resource
			var res fyne.Resource
			key := ""
			switch opt {
			case "Hit":
				if r, ok := ActionSprites["hit"]; ok {
					res = r
				} else if r, ok := ActionSprites["medium_hit"]; ok {
					res = r
				}
			case "Stamp":
				key = "stamp"
			case "Bend":
				key = "bend"
			case "Upset":
				key = "upset"
			case "Shrink":
				key = "shrink"
			case "Draw":
				key = "draw"
			}
			if key != "" {
				if r, ok := ActionSprites[key]; ok {
					res = r
				}
			}
			// create option button
			optBtn := widget.NewButtonWithIcon(opt, res, func() {
				// placeholder; will set in closure below
			})
			// override tapped behavior to capture opt variable
			o := opt
			optBtn.OnTapped = func() {
				cur = o
				lbl.SetText(cur)
				// update icon resource
				switch cur {
				case "", "Any":
					icon.Resource = nil
				case "Hit":
					if r, ok := ActionSprites["hit"]; ok {
						icon.Resource = r
					} else if r, ok := ActionSprites["medium_hit"]; ok {
						icon.Resource = r
					} else {
						icon.Resource = nil
					}
				default:
					// map as above
					mapKey := ""
					switch cur {
					case "Stamp":
						mapKey = "stamp"
					case "Bend":
						mapKey = "bend"
					case "Upset":
						mapKey = "upset"
					case "Shrink":
						mapKey = "shrink"
					case "Draw":
						mapKey = "draw"
					}
					if r, ok := ActionSprites[mapKey]; ok {
						icon.Resource = r
					} else {
						icon.Resource = nil
					}
				}
				icon.Refresh()
				lbl.Refresh()
				// hide popup if present and also hide the optsGrid
				if popup != nil && popup.Visible() {
					popup.Hide()
					optsGrid.Hide()
				}
				onChange(cur)
			}
			optsGrid.Add(optBtn)
		}

		// toggle behavior: show popup overlay anchored to this container
		toggleBtn := widget.NewButton("▼", func() {
			// popup created lazily below
		})

		// main container holds the display and toggle only; optsGrid will be shown in a PopUp overlay
		container := container.NewVBox(container.NewHBox(display, toggleBtn))

		// helper to set resource for current selection
		updateResource := func(sel string) {
			switch sel {
			case "", "Any":
				icon.Resource = nil
			case "Hit":
				if r, ok := ActionSprites["hit"]; ok {
					icon.Resource = r
				} else if r, ok := ActionSprites["medium_hit"]; ok {
					icon.Resource = r
				} else {
					icon.Resource = nil
				}
			default:
				mapKey := ""
				switch sel {
				case "Stamp":
					mapKey = "stamp"
				case "Bend":
					mapKey = "bend"
				case "Upset":
					mapKey = "upset"
				case "Shrink":
					mapKey = "shrink"
				case "Draw":
					mapKey = "draw"
				}
				if r, ok := ActionSprites[mapKey]; ok {
					icon.Resource = r
				} else {
					icon.Resource = nil
				}
			}
			icon.Refresh()
			lbl.Refresh()
		}

		// initial set
		lbl.SetText(initial)
		updateResource(initial)

		// position popup below the display when showing
		toggleBtn.OnTapped = func() {
			if popup != nil && popup.Visible() {
				popup.Hide()
				return
			}
			// lazy-create popup when we have a canvas; otherwise fallback to inline grid
			if popup == nil {
				canvasFor := fyne.CurrentApp().Driver().CanvasForObject(container)
				if canvasFor == nil {
					// fallback: toggle inline visibility
					if optsGrid.Visible() {
						optsGrid.Hide()
					} else {
						optsGrid.Show()
					}
					return
				}
				popup = widget.NewPopUp(optsGrid, canvasFor)
			}
			// ensure options are visible inside popup
			optsGrid.Show()
			// position popup under the display using relative position helper
			rel := fyne.NewPos(0, display.Size().Height+2)
			popup.ShowAtRelativePosition(rel, display)
		}

		return container, func() string { return cur }, func(s string) {
			cur = s
			lbl.SetText(cur)
			updateResource(cur)
			onChange(cur)
		}
	}

	// NOTE: dropdowns created after setIcon is defined so we can pass callbacks

	// result area will show icons horizontally
	resultBox := container.NewHBox()

	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapWord

	// try load sprites (order must match sprite layout)
	_ = LoadActionSprites("./assets/anvil-hits.png", []string{
		"weak_hit", "medium_hit", "strong_hit", "draw",
		"stamp", "bend", "upset", "shrink",
	})

	// no external icon setter needed; dropdown setter will update its own icon

	// create three icon dropdowns (they manage their own preview icon)
	selAObj, selAGet, selASet := makeIconDropdown("Any", func(s string) {})
	selBObj, selBGet, selBSet := makeIconDropdown("Any", func(s string) {})
	selCObj, selCGet, selCSet := makeIconDropdown("Any", func(s string) {})

	// (removed temporary textual debug output)

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
		sels := []string{selAGet(), selBGet(), selCGet()}
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
			resultBox.Objects = nil
			resultBox.Add(resultLabel)
			resultLabel.SetText(fmt.Sprintf("No solution: %v", err))
			resultBox.Refresh()
			return
		}

		// Build icon sequence
		resultBox.Objects = nil
		for i, a := range seq {
			key := ""
			switch a {
			case domain.WeakHit:
				key = "weak_hit"
			case domain.MediumHit:
				key = "medium_hit"
			case domain.StrongHit:
				key = "strong_hit"
			case domain.Draw:
				key = "draw"
			case domain.Stamp:
				key = "stamp"
			case domain.Bend:
				key = "bend"
			case domain.Upset:
				key = "upset"
			case domain.Shrink:
				key = "shrink"
			}
			var img *canvas.Image
			if r, ok := ActionSprites[key]; ok {
				img = canvas.NewImageFromResource(r)
			} else {
				img = canvas.NewImageFromResource(nil)
			}
			img.SetMinSize(fyne.NewSize(32, 32))
			img.FillMode = canvas.ImageFillContain
			// highlight last three with an outer border and inner background
			if i >= len(seq)-3 {
				outer := canvas.NewRectangle(color.NRGBA{R: 0xff, G: 0x99, B: 0x33, A: 0xff})
				outer.SetMinSize(fyne.NewSize(40, 40))
				inner := canvas.NewRectangle(color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff})
				inner.SetMinSize(fyne.NewSize(36, 36))
				wrapped := container.NewStack(outer, inner, container.NewCenter(img))
				resultBox.Add(wrapped)
			} else {
				resultBox.Add(img)
			}
			// textual debug removed
		}
		resultBox.Refresh()

	})

	// Layout
	form := container.NewVBox(
		widget.NewLabel("Anvil Forging"),
		widget.NewLabel("Target value (0..150):"),
		targetEntry,
		widget.NewLabel("Final three actions (in order, leave empty for Any):"),
		// place selectors close together
		container.NewHBox(selAObj, selBObj, selCObj),
		solveButton,
		widget.NewLabel("Result sequence (last three are final):"),
		resultBox,
	)

	// Initialize selects to Any (also updates preview icons)
	selASet("Any")
	selBSet("Any")
	selCSet("Any")

	// Preview grid showing each option with icon and text for verification
	preview := container.NewGridWithColumns(4)
	ordered := []struct {
		key   string
		label string
	}{
		{"weak_hit", "Hit, Light"},
		{"medium_hit", "Hit, Medium"},
		{"strong_hit", "Hit, Heavy"},
		{"draw", "Draw"},
		{"stamp", "Punch/Stamp"},
		{"bend", "Bend"},
		{"upset", "Upset"},
		{"shrink", "Shrink"},
	}
	for _, e := range ordered {
		var img *canvas.Image
		if r, ok := ActionSprites[e.key]; ok {
			img = canvas.NewImageFromResource(r)
		} else {
			img = canvas.NewImageFromResource(nil)
		}
		img.SetMinSize(fyne.NewSize(24, 24))
		lbl := widget.NewLabel(e.label)
		preview.Add(container.NewHBox(img, lbl))
	}

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
