package puzzle

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestGridSizeTable(t *testing.T) {
	cases := []struct{ players, want int }{
		{1, 3}, {9, 3},
		{10, 4}, {16, 4},
		{17, 5}, {25, 5},
		{26, 6}, {36, 6},
		{37, 7}, {49, 7},
		{50, 8}, {64, 8},
	}
	for _, tc := range cases {
		if got := GridSize(tc.players); got != tc.want {
			t.Errorf("GridSize(%d) = %d, want %d", tc.players, got, tc.want)
		}
	}
}

func TestSegmentIDMapping(t *testing.T) {
	if got := SegmentID(Pos{X: 0, Y: 0}); got != "segment_a1" {
		t.Errorf("SegmentID({0,0}) = %q", got)
	}
	if got := SegmentID(Pos{X: 2, Y: 2}); got != "segment_c3" {
		t.Errorf("SegmentID({2,2}) = %q", got)
	}

	for gridSize := 3; gridSize <= 8; gridSize++ {
		ids := AllSegmentIDs(gridSize)
		if len(ids) != gridSize*gridSize {
			t.Fatalf("AllSegmentIDs(%d) returned %d ids", gridSize, len(ids))
		}
		for i, id := range ids {
			p, err := ParseSegmentID(id, gridSize)
			if err != nil {
				t.Fatalf("ParseSegmentID(%q): %v", id, err)
			}
			if want := (Pos{X: i % gridSize, Y: i / gridSize}); p != want {
				t.Errorf("ParseSegmentID(%q) = %+v, want %+v", id, p, want)
			}
		}
	}
}

func TestParseSegmentIDRejectsMalformed(t *testing.T) {
	bad := []string{"", "segment_", "seg_a1", "segment_1a", "segment_a0", "segment_d1", "segment_a4", "segment_aa"}
	for _, id := range bad {
		if _, err := ParseSegmentID(id, 3); err == nil {
			t.Errorf("ParseSegmentID(%q, 3) accepted", id)
		}
	}
}

// testImage builds a w×h image where every pixel encodes its coordinates:
// R = x, G = y. Lets tile tests verify exact source regions.
func testImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), A: 0xff})
		}
	}
	return img
}

func decodeTile(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("tile does not decode: %v", err)
	}
	return img
}

func TestGenerateTilesSquareSource(t *testing.T) {
	tiles, err := GenerateTiles(testImage(96, 96), 3)
	if err != nil {
		t.Fatalf("GenerateTiles: %v", err)
	}
	if len(tiles) != 9 {
		t.Fatalf("got %d tiles, want 9", len(tiles))
	}

	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			id := SegmentID(Pos{X: x, Y: y})
			img := decodeTile(t, tiles[id])
			if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
				t.Errorf("%s is %dx%d, want 32x32", id, img.Bounds().Dx(), img.Bounds().Dy())
			}
			r, g, _, _ := img.At(0, 0).RGBA()
			if uint8(r>>8) != uint8(32*x) || uint8(g>>8) != uint8(32*y) {
				t.Errorf("%s top-left = (%d,%d), want (%d,%d)", id, r>>8, g>>8, 32*x, 32*y)
			}
		}
	}
}

// TestGenerateTilesCenterCrop verifies a non-square source is center-cropped:
// a 120×96 image crops 12px off each horizontal side.
func TestGenerateTilesCenterCrop(t *testing.T) {
	tiles, err := GenerateTiles(testImage(120, 96), 3)
	if err != nil {
		t.Fatalf("GenerateTiles: %v", err)
	}
	img := decodeTile(t, tiles["segment_a1"])
	r, g, _, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != 12 || uint8(g>>8) != 0 {
		t.Errorf("crop origin pixel = (%d,%d), want (12,0)", r>>8, g>>8)
	}
}

// TestGenerateTilesUnevenDivision proves all source pixels are covered when
// the side is not divisible by the grid size (100 = 3*33+1).
func TestGenerateTilesUnevenDivision(t *testing.T) {
	tiles, err := GenerateTiles(testImage(100, 100), 3)
	if err != nil {
		t.Fatalf("GenerateTiles: %v", err)
	}
	total := 0
	for _, b := range tiles {
		img := decodeTile(t, b)
		total += img.Bounds().Dx() * img.Bounds().Dy()
	}
	if total != 100*100 {
		t.Errorf("tiles cover %d pixels, want %d", total, 100*100)
	}
}

func TestGenerateTilesTooSmall(t *testing.T) {
	if _, err := GenerateTiles(testImage(2, 2), 3); err == nil {
		t.Error("accepted an image smaller than the grid")
	}
}
