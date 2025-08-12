package models

import (
	"fmt"
	"math/rand"
	"time"
)

// Position represents a grid position
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Fragment represents a puzzle fragment on the central grid
type Fragment struct {
	ID              string    `json:"fragmentId"`
	SegmentID       string    `json:"segmentId"`
	PlayerID        string    `json:"playerId,omitempty"`
	Position        Position  `json:"position"`
	CorrectPosition Position  `json:"correctPosition"`
	Visible         bool      `json:"visible"`
	LastMoved       time.Time `json:"lastMoved,omitempty"`
	MoveCount       int       `json:"moveCount"`
}

// IsCorrect checks if the fragment is in its correct position
func (f *Fragment) IsCorrect() bool {
	return f.Position.X == f.CorrectPosition.X && f.Position.Y == f.CorrectPosition.Y
}

// IsOwned checks if the fragment is owned by a player
func (f *Fragment) IsOwned() bool {
	return f.PlayerID != ""
}

// PuzzleGrid represents the central puzzle grid
type PuzzleGrid struct {
	Size      int                  `json:"size"`
	Fragments map[string]*Fragment `json:"fragments"`
	Grid      [][]*Fragment        `json:"-"` // 2D array for easy position lookup
}

// NewPuzzleGrid creates a new puzzle grid
func NewPuzzleGrid(size int) *PuzzleGrid {
	pg := &PuzzleGrid{
		Size:      size,
		Fragments: make(map[string]*Fragment),
		Grid:      make([][]*Fragment, size),
	}

	// Initialize 2D grid
	for i := range pg.Grid {
		pg.Grid[i] = make([]*Fragment, size)
	}

	return pg
}

// AddFragment adds a fragment to the grid at a random position
func (pg *PuzzleGrid) AddFragment(segmentID string, playerID string) *Fragment {
	// Generate fragment ID
	fragmentID := fmt.Sprintf("fragment_%s", playerID)
	if playerID == "" {
		fragmentID = fmt.Sprintf("fragment_%s_%d", segmentID, time.Now().UnixNano())
	}

	// Calculate correct position based on segment ID (e.g., "A1", "B2", etc.)
	correctPos := pg.getCorrectPositionFromSegmentID(segmentID)

	// Find random empty position
	pos := pg.findRandomEmptyPosition()

	fragment := &Fragment{
		ID:              fragmentID,
		SegmentID:       segmentID,
		PlayerID:        playerID,
		Position:        pos,
		CorrectPosition: correctPos,
		Visible:         true,
		LastMoved:       time.Now(),
		MoveCount:       0,
	}

	pg.Fragments[fragmentID] = fragment
	pg.Grid[pos.Y][pos.X] = fragment

	return fragment
}

// getCorrectPositionFromSegmentID converts segment ID to grid position
func (pg *PuzzleGrid) getCorrectPositionFromSegmentID(segmentID string) Position {
	if len(segmentID) < 2 {
		return Position{X: 0, Y: 0}
	}

	// Extract row letter and column number
	rowLetter := segmentID[0]
	colNumber := segmentID[1] - '1' // Convert '1' to 0, '2' to 1, etc.

	row := int(rowLetter - 'A') // Convert 'A' to 0, 'B' to 1, etc.
	col := int(colNumber)

	return Position{X: col, Y: row}
}

// findRandomEmptyPosition finds a random empty position on the grid
func (pg *PuzzleGrid) findRandomEmptyPosition() Position {
	// Collect all empty positions
	emptyPositions := []Position{}
	for y := 0; y < pg.Size; y++ {
		for x := 0; x < pg.Size; x++ {
			if pg.Grid[y][x] == nil {
				emptyPositions = append(emptyPositions, Position{X: x, Y: y})
			}
		}
	}

	// If no empty positions (shouldn't happen), return 0,0
	if len(emptyPositions) == 0 {
		return Position{X: 0, Y: 0}
	}

	// Return random empty position
	return emptyPositions[rand.Intn(len(emptyPositions))]
}

// SwapFragments swaps two fragments on the grid
func (pg *PuzzleGrid) SwapFragments(fragment1ID, fragment2ID string) error {
	frag1, ok1 := pg.Fragments[fragment1ID]
	frag2, ok2 := pg.Fragments[fragment2ID]

	if !ok1 || !ok2 {
		return fmt.Errorf("fragment not found")
	}

	// Swap positions in grid
	pg.Grid[frag1.Position.Y][frag1.Position.X] = frag2
	pg.Grid[frag2.Position.Y][frag2.Position.X] = frag1

	// Swap position values
	frag1.Position, frag2.Position = frag2.Position, frag1.Position

	// Update move metadata
	now := time.Now()
	frag1.LastMoved = now
	frag1.MoveCount++
	frag2.LastMoved = now
	frag2.MoveCount++

	return nil
}

// MoveFragment moves a fragment to an empty position
func (pg *PuzzleGrid) MoveFragment(fragmentID string, newPos Position) error {
	// Validate position
	if newPos.X < 0 || newPos.X >= pg.Size || newPos.Y < 0 || newPos.Y >= pg.Size {
		return fmt.Errorf("position out of bounds")
	}

	fragment, ok := pg.Fragments[fragmentID]
	if !ok {
		return fmt.Errorf("fragment not found")
	}

	// Check if target position is empty
	if pg.Grid[newPos.Y][newPos.X] != nil {
		return fmt.Errorf("target position is not empty")
	}

	// Clear old position
	pg.Grid[fragment.Position.Y][fragment.Position.X] = nil

	// Set new position
	fragment.Position = newPos
	pg.Grid[newPos.Y][newPos.X] = fragment

	// Update move metadata
	fragment.LastMoved = time.Now()
	fragment.MoveCount++

	return nil
}

// CheckCompletion checks if all fragments are in correct positions
func (pg *PuzzleGrid) CheckCompletion() bool {
	// First check if all positions are filled
	for y := 0; y < pg.Size; y++ {
		for x := 0; x < pg.Size; x++ {
			if pg.Grid[y][x] == nil {
				return false
			}
		}
	}

	// Then check if all fragments are in correct positions
	for _, fragment := range pg.Fragments {
		if !fragment.IsCorrect() {
			return false
		}
	}

	return true
}

// GetFragmentAt returns the fragment at a specific position
func (pg *PuzzleGrid) GetFragmentAt(pos Position) *Fragment {
	if pos.X < 0 || pos.X >= pg.Size || pos.Y < 0 || pos.Y >= pg.Size {
		return nil
	}
	return pg.Grid[pos.Y][pos.X]
}

// GetGuideHighlights returns positions where a fragment could go (for guide tokens)
func (pg *PuzzleGrid) GetGuideHighlights(fragmentID string, reductionLevel int) []Position {
	fragment, ok := pg.Fragments[fragmentID]
	if !ok {
		return []Position{}
	}

	// Start with all positions
	positions := []Position{}
	for y := 0; y < pg.Size; y++ {
		for x := 0; x < pg.Size; x++ {
			positions = append(positions, Position{X: x, Y: y})
		}
	}

	// Always include the correct position
	correctPos := fragment.CorrectPosition

	// Calculate how many positions to show based on reduction level
	totalPositions := pg.Size * pg.Size
	positionsToRemove := reductionLevel * (totalPositions / 7)
	positionsToShow := totalPositions - positionsToRemove

	if positionsToShow < 1 {
		positionsToShow = 1 // Always show at least the correct position
	}

	// If we're showing very few positions, prioritize around correct position
	if positionsToShow <= 9 {
		highlights := []Position{correctPos}

		// Add adjacent positions
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				newPos := Position{X: correctPos.X + dx, Y: correctPos.Y + dy}
				if newPos.X >= 0 && newPos.X < pg.Size && newPos.Y >= 0 && newPos.Y < pg.Size {
					highlights = append(highlights, newPos)
					if len(highlights) >= positionsToShow {
						return highlights
					}
				}
			}
		}
		return highlights
	}

	// For more positions, randomly select but always include correct position
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	// Ensure correct position is included
	result := []Position{correctPos}
	for _, pos := range positions {
		if pos.X == correctPos.X && pos.Y == correctPos.Y {
			continue // Already included
		}
		result = append(result, pos)
		if len(result) >= positionsToShow {
			break
		}
	}

	return result
}

// MoveRecommendation represents a fragment swap recommendation
type MoveRecommendation struct {
	ID             string    `json:"moveId"`
	FromPlayerID   string    `json:"fromPlayerId"`
	FromPlayerName string    `json:"fromPlayerName"`
	ToPlayerID     string    `json:"toPlayerId"`
	FromFragmentID string    `json:"fromFragmentId"`
	ToFragmentID   string    `json:"toFragmentId"`
	Reasoning      string    `json:"reasoning"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	Status         string    `json:"status"` // pending, accepted, rejected, expired
}
