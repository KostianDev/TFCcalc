package adapterui

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// sprite layout: 4 cols x 2 rows (left-to-right, top-to-bottom)
const spriteCols = 4
const spriteRows = 2

// ActionSprites holds canvas resources for each action.
var ActionSprites map[string]*fyne.StaticResource

// LoadActionSprites slices the sprite image and populates ActionSprites.
func LoadActionSprites(path string, actionOrder []string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	tileW := bounds.Dx() / spriteCols
	tileH := bounds.Dy() / spriteRows

	ActionSprites = make(map[string]*fyne.StaticResource)

	for idx, name := range actionOrder {
		col := idx % spriteCols
		row := idx / spriteCols
		x0 := bounds.Min.X + col*tileW
		y0 := bounds.Min.Y + row*tileH
		rect := image.Rect(x0, y0, x0+tileW, y0+tileH)
		sub := img.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(rect)

		var buf bytes.Buffer
		if err := png.Encode(&buf, sub); err != nil {
			return err
		}
		res := fyne.NewStaticResource(name+".png", buf.Bytes())
		ActionSprites[name] = res
	}

	// also try to load a standalone hit.png (generic Hit icon) if present
	dir := filepath.Dir(path)
	hitPath := filepath.Join(dir, "hit.png")
	if _, err := os.Stat(hitPath); err == nil {
		if b, err := os.ReadFile(hitPath); err == nil {
			// register as key "hit"
			res := fyne.NewStaticResource("hit.png", b)
			ActionSprites["hit"] = res
		}
	}
	return nil
}

// NewSpriteImage returns a canvas.Image for given action key (or nil if missing).
func NewSpriteImage(action string) *canvas.Image {
	if ActionSprites == nil {
		return nil
	}
	if r, ok := ActionSprites[action]; ok {
		img := canvas.NewImageFromResource(r)
		img.SetMinSize(fyne.NewSize(32, 32))
		img.FillMode = canvas.ImageFillContain
		return img
	}
	return nil
}
