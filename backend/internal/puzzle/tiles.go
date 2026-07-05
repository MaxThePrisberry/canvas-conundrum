package puzzle

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
)

// GenerateTiles center-crops the source image to a square and slices it into
// gridSize² PNG tiles keyed by segment ID. Called once per game at the start
// of puzzle preparation; the result lives only in memory (game-design.md
// § Asset Delivery). Tile boundaries are computed as i*side/gridSize so every
// source pixel of the crop is covered even when side % gridSize != 0.
func GenerateTiles(src image.Image, gridSize int) (map[string][]byte, error) {
	if gridSize < 1 {
		return nil, fmt.Errorf("invalid grid size %d", gridSize)
	}
	square := centerCrop(src)
	side := square.Bounds().Dx()
	if side < gridSize {
		return nil, fmt.Errorf("image side %dpx smaller than grid size %d", side, gridSize)
	}

	tiles := make(map[string][]byte, gridSize*gridSize)
	origin := square.Bounds().Min
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			r := image.Rect(
				origin.X+x*side/gridSize,
				origin.Y+y*side/gridSize,
				origin.X+(x+1)*side/gridSize,
				origin.Y+(y+1)*side/gridSize,
			)
			tile := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
			draw.Draw(tile, tile.Bounds(), square, r.Min, draw.Src)

			buf, err := encodePNG(tile)
			if err != nil {
				return nil, err
			}
			tiles[SegmentID(Pos{X: x, Y: y})] = buf
		}
	}
	return tiles, nil
}

// EncodePNG renders any image as PNG bytes (used for the full-image clarity
// preview).
func EncodePNG(img image.Image) ([]byte, error) {
	return encodePNG(img)
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode tile: %w", err)
	}
	return buf.Bytes(), nil
}

// centerCrop returns the largest centered square view of img.
func centerCrop(img image.Image) image.Image {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	x0 := b.Min.X + (b.Dx()-side)/2
	y0 := b.Min.Y + (b.Dy()-side)/2
	crop := image.Rect(x0, y0, x0+side, y0+side)

	out := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(out, out.Bounds(), img, crop.Min, draw.Src)
	return out
}
