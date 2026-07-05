// Package puzzle implements grid geometry (game-design.md § Dynamic Grid
// System, § Segment ID Convention) and runtime tile generation.
package puzzle

import (
	"fmt"
	"strconv"
)

// Pos is a 0-based grid coordinate, origin top-left: x = column, y = row.
type Pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// GridSize returns the central grid dimension for a player count, per the
// scaling table in game-design.md. Player counts above 64 are unsupported
// (config validation caps maxPlayers).
func GridSize(players int) int {
	switch {
	case players <= 9:
		return 3
	case players <= 16:
		return 4
	case players <= 25:
		return 5
	case players <= 36:
		return 6
	case players <= 49:
		return 7
	default:
		return 8
	}
}

// SegmentID renders a position as its canonical segment identifier:
// segment_{row letter}{column number}, e.g. {0,0} → segment_a1.
func SegmentID(p Pos) string {
	return fmt.Sprintf("segment_%c%d", 'a'+rune(p.Y), p.X+1)
}

// ParseSegmentID inverts SegmentID. gridSize bounds the accepted
// coordinates; anything outside or malformed is an error.
func ParseSegmentID(id string, gridSize int) (Pos, error) {
	const prefix = "segment_"
	if len(id) < len(prefix)+2 || id[:len(prefix)] != prefix {
		return Pos{}, fmt.Errorf("malformed segment id %q", id)
	}
	rest := id[len(prefix):]
	row := int(rest[0] - 'a')
	col, err := strconv.Atoi(rest[1:])
	if err != nil {
		return Pos{}, fmt.Errorf("malformed segment id %q", id)
	}
	p := Pos{X: col - 1, Y: row}
	if p.X < 0 || p.X >= gridSize || p.Y < 0 || p.Y >= gridSize {
		return Pos{}, fmt.Errorf("segment id %q outside %dx%d grid", id, gridSize, gridSize)
	}
	return p, nil
}

// CorrectPos is the cell a segment belongs on when the puzzle is solved —
// the identity mapping of ParseSegmentID, named for intent at call sites.
func CorrectPos(segmentID string, gridSize int) (Pos, error) {
	return ParseSegmentID(segmentID, gridSize)
}

// AllSegmentIDs enumerates every segment of a gridSize×gridSize puzzle in
// row-major order.
func AllSegmentIDs(gridSize int) []string {
	ids := make([]string, 0, gridSize*gridSize)
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			ids = append(ids, SegmentID(Pos{X: x, Y: y}))
		}
	}
	return ids
}
