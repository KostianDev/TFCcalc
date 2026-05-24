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
	// Target input (slider 0..150)
	targetSlider := widget.NewSlider(0, 150)
	targetSlider.Step = 1
	targetSlider.Value = 0 // default value

	targetValueLabel := widget.NewLabel("0")
	targetValueLabel.Alignment = fyne.TextAlignCenter
	targetValueLabel.TextStyle = fyne.TextStyle{Bold: true}

	targetSlider.OnChanged = func(val float64) {
		targetValueLabel.SetText(strconv.Itoa(int(val)))
	}

	targetContainer := container.NewBorder(nil, nil, nil, targetValueLabel, targetSlider)

	// try load sprites (order must match sprite layout)
	_ = LoadActionSprites("./assets/anvil-hits.png", []string{
		"weak_hit", "medium_hit", "strong_hit", "draw",
		"stamp", "bend", "upset", "shrink",
	})

	getPaletteIcon := func(sel string) fyne.Resource {
		switch sel {
		case "Hit":
			if r, ok := ActionSprites["hit"]; ok {
				return r
			}
			return ActionSprites["medium_hit"]
		case "Stamp":
			return ActionSprites["stamp"]
		case "Bend":
			return ActionSprites["bend"]
		case "Upset":
			return ActionSprites["upset"]
		case "Shrink":
			return ActionSprites["shrink"]
		case "Draw":
			return ActionSprites["draw"]
		}
		return nil
	}

	activeSlot := 0
	slotVals := []string{"Any", "Any", "Any"}

	type slotObj struct {
		bg  *canvas.Rectangle
		img *canvas.Image
		btn *widget.Button
		box *fyne.Container
	}
	var slots [3]*slotObj

	var refreshSlots func()

	slotsRow := container.NewHBox()
	for i := 0; i < 3; i++ {
		s := &slotObj{}
		s.bg = canvas.NewRectangle(color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff})
		s.bg.StrokeColor = color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}
		s.bg.StrokeWidth = 2
		s.bg.SetMinSize(fyne.NewSize(48, 48))
		s.img = canvas.NewImageFromResource(nil)
		s.img.SetMinSize(fyne.NewSize(32, 32))
		s.img.FillMode = canvas.ImageFillContain

		idx := i
		s.btn = widget.NewButton("", func() {
			activeSlot = idx
			refreshSlots()
		})
		s.btn.Importance = widget.LowImportance

		s.box = container.NewStack(s.bg, container.NewCenter(s.img), s.btn)
		slots[i] = s
		slotsRow.Add(s.box)
	}

	refreshSlots = func() {
		for i := 0; i < 3; i++ {
			if i == activeSlot {
				slots[i].bg.StrokeColor = color.NRGBA{R: 0xff, G: 0x99, B: 0x33, A: 0xff}
				slots[i].bg.FillColor = color.NRGBA{R: 0x55, G: 0x44, B: 0x33, A: 0xff} // Active slot highlight
			} else {
				slots[i].bg.StrokeColor = color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}
				slots[i].bg.FillColor = color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff}
			}
			slots[i].bg.Refresh()

			res := getPaletteIcon(slotVals[i])
			if res == nil {
				slots[i].img.Hide()
			} else {
				slots[i].img.Resource = res
				slots[i].img.Show()
			}
			slots[i].img.Refresh()
		}
	}
	// Initial refresh
	refreshSlots()

	// Palette of actions
	paletteOptions := []string{"Any", "Hit", "Stamp", "Bend", "Upset", "Shrink", "Draw"}
	paletteGrid := container.NewGridWrap(fyne.NewSize(120, 40))
	for _, opt := range paletteOptions {
		o := opt
		pBtn := widget.NewButtonWithIcon(o, getPaletteIcon(o), func() {
			slotVals[activeSlot] = o
			activeSlot = (activeSlot + 1) % 3
			refreshSlots()
		})
		paletteGrid.Add(pBtn)
	}

	// Result area
	resultBox := container.NewHBox()
	resultScroll := container.NewHScroll(resultBox)
	resultScroll.SetMinSize(fyne.NewSize(0, 120)) // assure enough height

	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapWord

	solveButton := widget.NewButton("Solve", func() {
		resultLabel.SetText("")
		resultBox.Objects = nil

		tval := int(targetSlider.Value)
		if tval < 0 || tval > 150 {
			resultLabel.SetText("Enter a valid integer target between 0 and 150.")
			resultBox.Add(resultLabel)
			resultBox.Refresh()
			return
		}

		var pattern [3]domain.FinalRequirement
		for i, s := range slotVals {
			switch s {
			case "", "Any":
				pattern[i] = domain.FinalRequirement{Kind: domain.FinalAny}
			case "Hit":
				pattern[i] = domain.FinalRequirement{Kind: domain.FinalHit}
			default:
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

		seq, err := svc.SolveForTarget(tval, pattern)
		if err != nil {
			resultBox.Add(resultLabel)
			resultLabel.SetText(fmt.Sprintf("No solution: %v", err))
			resultBox.Refresh()
			return
		}

		currentValue := 0
		for i, a := range seq {
			delta := domain.ForgeActionDelta[a]
			currentValue += delta

			deltaStr := fmt.Sprintf("%+d", delta)
			deltaLbl := canvas.NewText(deltaStr, color.NRGBA{R: 170, G: 170, B: 170, A: 255})
			deltaLbl.TextSize = 12
			deltaLbl.Alignment = fyne.TextAlignCenter

			valLbl := canvas.NewText(strconv.Itoa(currentValue), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			valLbl.TextSize = 14
			valLbl.Alignment = fyne.TextAlignCenter
			if currentValue < 0 || currentValue > 150 {
				valLbl.Color = color.NRGBA{R: 255, G: 100, B: 100, A: 255} // Highlight out of bounds intermediate
			}

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

			var iconLayout fyne.CanvasObject
			if i >= len(seq)-3 {
				outer := canvas.NewRectangle(color.NRGBA{R: 0xff, G: 0x99, B: 0x33, A: 0xff})
				outer.SetMinSize(fyne.NewSize(40, 40))
				inner := canvas.NewRectangle(color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff})
				inner.SetMinSize(fyne.NewSize(36, 36))
				iconLayout = container.NewStack(outer, inner, container.NewCenter(img))
			} else {
				iconLayout = container.NewCenter(img)
			}

			col := container.NewVBox(deltaLbl, iconLayout, valLbl)
			resultBox.Add(col)
		}
		resultBox.Refresh()
	})

	resetBtn := widget.NewButton("Reset", func() {
		targetSlider.SetValue(0)
		activeSlot = 0
		slotVals[0], slotVals[1], slotVals[2] = "Any", "Any", "Any"
		refreshSlots()
		resultBox.Objects = nil
		resultLabel.SetText("")
		resultBox.Refresh()
	})

	btnBox := container.NewHBox(solveButton, resetBtn)

	// Layout
	form := container.NewVBox(
		widget.NewLabel("Anvil Forging"),
		widget.NewLabel("Target value:"),
		targetContainer,
		widget.NewLabel("Final three actions (click slot to select, then pick from palette):"),
		slotsRow,
		paletteGrid,
		btnBox,
		widget.NewLabel("Result sequence (last three are final):"),
		resultScroll, // scrollable area for sequence
	)

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
