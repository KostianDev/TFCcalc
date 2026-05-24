package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// Render lineInfo into colored monospace text segments.

// palette is the set of colors to cycle through for different depths.
var palette = []color.Color{
	color.RGBA{R: 255, G: 102, B: 102, A: 255}, // Light Red
	color.RGBA{R: 102, G: 255, B: 102, A: 255}, // Light Green
	color.RGBA{R: 102, G: 178, B: 255, A: 255}, // Light Blue
	color.RGBA{R: 255, G: 255, B: 102, A: 255}, // Light Yellow
	color.RGBA{R: 255, G: 153, B: 255, A: 255}, // Light Pink
	color.RGBA{R: 153, G: 255, B: 255, A: 255}, // Light Cyan
}

// RenderLines returns a VBox that lays out each line as colored text segments.
func RenderLines(lines []lineInfo) *fyne.Container {
	box := container.NewVBox()

	for _, ln := range lines {
		var segments []fyne.CanvasObject
		depth := len(ln.PrefixParts) - 1

		// Draw vertical bars or spaces for each ancestor level.
		for lvl := 0; lvl < depth; lvl++ {
			if ln.PrefixParts[lvl] {
				// If the ancestor was the last child, draw spaces.
				txt := canvas.NewText("    ", color.White)
				txt.TextStyle = fyne.TextStyle{Monospace: true}
				segments = append(segments, txt)
			} else {
				// Otherwise draw a vertical bar in the color for this level.
				txt := canvas.NewText("│   ", palette[lvl%len(palette)])
				txt.TextStyle = fyne.TextStyle{Monospace: true}
				segments = append(segments, txt)
			}
		}

		// Draw branch symbol in the color at current depth.
		branchSymbol := "├── "
		if ln.IsLast {
			branchSymbol = "└── "
		}
		brText := canvas.NewText(branchSymbol, palette[depth%len(palette)])
		brText.TextStyle = fyne.TextStyle{Monospace: true}
		segments = append(segments, brText)

		// Draw the node text in the same color.
		nodeTxt := canvas.NewText(ln.Text, palette[depth%len(palette)])
		nodeTxt.TextStyle = fyne.TextStyle{Monospace: true}
		segments = append(segments, nodeTxt)

		// Combine into an HBox and add to the VBox.
		box.Add(container.NewHBox(segments...))
	}

	return box
}
