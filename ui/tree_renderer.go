// ui/tree_renderer.go
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

//
// This file knows how to take a []lineInfo (from tree_formatter.go) and turn it into
// a Fyne container full of colored canvas.Text segments. We draw vertical bars “│   ”,
// branch symbols “├── ” / “└── ”, and the node text itself in a monospace style.
//
// - palette: array of colors (cycled by depth)
// - RenderLines: builds a *fyne.Container (VBox) by concatenating HBoxes
//

// palette is the set of distinct colors to cycle through for different depths.
var palette = []color.Color{
	color.RGBA{R: 255, G: 102, B: 102, A: 255}, // Light Red
	color.RGBA{R: 102, G: 255, B: 102, A: 255}, // Light Green
	color.RGBA{R: 102, G: 178, B: 255, A: 255}, // Light Blue
	color.RGBA{R: 255, G: 255, B: 102, A: 255}, // Light Yellow
	color.RGBA{R: 255, G: 153, B: 255, A: 255}, // Light Pink
	color.RGBA{R: 153, G: 255, B: 255, A: 255}, // Light Cyan
}

// RenderLines accepts a slice of lineInfo and returns a *fyne.Container (VBox)
// that lays out each line as an HBox of canvas.Text segments with the correct colors.
//
// Each line is composed of:
//  1. “│   ” or “    ” segments for each ancestor level
//  2. “├── ” or “└── ” branch symbol at the current depth
//  3. The node text itself (e.g. “Copper (221.25mB | 2.212Ing)”).
func RenderLines(lines []lineInfo) *fyne.Container {
	box := container.NewVBox()

	for _, ln := range lines {
		var segments []fyne.CanvasObject
		depth := len(ln.PrefixParts) - 1

		// 1) Draw vertical bars or spaces for each ancestor level.
		for lvl := 0; lvl < depth; lvl++ {
			if ln.PrefixParts[lvl] {
				// If the ancestor at this level was the last child, draw spaces “    ”.
				txt := canvas.NewText("    ", color.White)
				txt.TextStyle = fyne.TextStyle{Monospace: true}
				segments = append(segments, txt)
			} else {
				// Otherwise draw a vertical bar “│   ” in the color for this level.
				txt := canvas.NewText("│   ", palette[lvl%len(palette)])
				txt.TextStyle = fyne.TextStyle{Monospace: true}
				segments = append(segments, txt)
			}
		}

		// 2) Draw branch symbol “├── ” or “└── ” in the color at current depth.
		branchSymbol := "├── "
		if ln.IsLast {
			branchSymbol = "└── "
		}
		brText := canvas.NewText(branchSymbol, palette[depth%len(palette)])
		brText.TextStyle = fyne.TextStyle{Monospace: true}
		segments = append(segments, brText)

		// 3) Draw the node’s text in the same color.
		nodeTxt := canvas.NewText(ln.Text, palette[depth%len(palette)])
		nodeTxt.TextStyle = fyne.TextStyle{Monospace: true}
		segments = append(segments, nodeTxt)

		// 4) Combine into an HBox and add to the VBox.
		box.Add(container.NewHBox(segments...))
	}

	return box
}
