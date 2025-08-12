package services

import (
	"canvas-conundrum/config"
	"canvas-conundrum/constants"
	"canvas-conundrum/models"
	"canvas-conundrum/utils"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// PuzzleService manages puzzle assembly logic
type PuzzleService struct {
	mu                 sync.RWMutex
	segmentAssignments map[string]string // playerID -> segmentID
	unassignedSegments []string
	recommendations    map[string]*models.MoveRecommendation
}

// NewPuzzleService creates a new puzzle service
func NewPuzzleService() *PuzzleService {
	return &PuzzleService{
		segmentAssignments: make(map[string]string),
		recommendations:    make(map[string]*models.MoveRecommendation),
	}
}

// AssignSegments assigns puzzle segments to players
func (ps *PuzzleService) AssignSegments(players map[string]*models.Player, gridSize int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Clear previous assignments
	ps.segmentAssignments = make(map[string]string)
	ps.unassignedSegments = []string{}

	// Generate all segment IDs for the grid
	allSegments := ps.generateSegmentIDs(gridSize)

	// Shuffle segments for random assignment
	rand.Shuffle(len(allSegments), func(i, j int) {
		allSegments[i], allSegments[j] = allSegments[j], allSegments[i]
	})

	// Assign segments to connected players
	segmentIndex := 0
	for playerID, player := range players {
		if player.IsActive && segmentIndex < len(allSegments) {
			segmentID := allSegments[segmentIndex]
			ps.segmentAssignments[playerID] = segmentID
			player.AssignedSegment = segmentID
			segmentIndex++
			log.Printf("Assigned segment %s to player %s", segmentID, playerID)
		}
	}

	// Remaining segments are unassigned
	for i := segmentIndex; i < len(allSegments); i++ {
		ps.unassignedSegments = append(ps.unassignedSegments, allSegments[i])
	}

	log.Printf("Puzzle segments assigned: %d to players, %d unassigned",
		segmentIndex, len(ps.unassignedSegments))
}

// generateSegmentIDs generates segment IDs based on grid size
func (ps *PuzzleService) generateSegmentIDs(gridSize int) []string {
	segments := []string{}
	for row := 0; row < gridSize; row++ {
		for col := 0; col < gridSize; col++ {
			// Generate ID like "A1", "B2", etc.
			rowLetter := string(rune('A' + row))
			colNumber := col + 1
			segmentID := fmt.Sprintf("%s%d", rowLetter, colNumber)
			segments = append(segments, segmentID)
		}
	}
	return segments
}

// GetSegmentAssignment returns the segment assigned to a player
func (ps *PuzzleService) GetSegmentAssignment(playerID string) (string, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	segment, exists := ps.segmentAssignments[playerID]
	return segment, exists
}

// GetAllAssignments returns all segment assignments
func (ps *PuzzleService) GetAllAssignments() map[string]string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	// Return a copy to avoid race conditions
	assignments := make(map[string]string)
	for k, v := range ps.segmentAssignments {
		assignments[k] = v
	}
	return assignments
}

// GetUnassignedSegments returns list of unassigned segments
func (ps *PuzzleService) GetUnassignedSegments() []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	// Return a copy
	segments := make([]string, len(ps.unassignedSegments))
	copy(segments, ps.unassignedSegments)
	return segments
}

// GetNextUnassignedSegment gets the next unassigned segment for auto-completion
func (ps *PuzzleService) GetNextUnassignedSegment() (string, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if len(ps.unassignedSegments) == 0 {
		return "", false
	}

	segment := ps.unassignedSegments[0]
	ps.unassignedSegments = ps.unassignedSegments[1:]
	return segment, true
}

// CreateRecommendation creates a new move recommendation
func (ps *PuzzleService) CreateRecommendation(fromPlayerID, fromPlayerName, toPlayerID,
	fromFragmentID, toFragmentID, reasoning string) (*models.MoveRecommendation, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	recommendation := &models.MoveRecommendation{
		ID:             utils.GenerateMoveID(),
		FromPlayerID:   fromPlayerID,
		FromPlayerName: fromPlayerName,
		ToPlayerID:     toPlayerID,
		FromFragmentID: fromFragmentID,
		ToFragmentID:   toFragmentID,
		Reasoning:      reasoning,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(30 * time.Second), // 30 second expiry
		Status:         "pending",
	}

	ps.recommendations[recommendation.ID] = recommendation

	log.Printf("Recommendation created: %s -> %s (fragments: %s <-> %s)",
		fromPlayerID, toPlayerID, fromFragmentID, toFragmentID)

	return recommendation, nil
}

// GetRecommendation retrieves a recommendation by ID
func (ps *PuzzleService) GetRecommendation(recommendationID string) (*models.MoveRecommendation, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	rec, exists := ps.recommendations[recommendationID]
	return rec, exists
}

// UpdateRecommendationStatus updates the status of a recommendation
func (ps *PuzzleService) UpdateRecommendationStatus(recommendationID string, status string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	rec, exists := ps.recommendations[recommendationID]
	if !exists {
		return fmt.Errorf("recommendation not found")
	}

	rec.Status = status
	log.Printf("Recommendation %s status updated to: %s", recommendationID, status)
	return nil
}

// ExpireRecommendation marks a recommendation as expired
func (ps *PuzzleService) ExpireRecommendation(recommendationID string, reason string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if rec, exists := ps.recommendations[recommendationID]; exists {
		rec.Status = "expired"
		log.Printf("Recommendation %s expired: %s", recommendationID, reason)
	}
}

// CleanupExpiredRecommendations removes expired recommendations
func (ps *PuzzleService) CleanupExpiredRecommendations() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	for id, rec := range ps.recommendations {
		if now.After(rec.ExpiresAt) && rec.Status == "pending" {
			rec.Status = "expired"
			log.Printf("Auto-expiring recommendation %s", id)
		}
	}
}

// CalculateGuideHighlights calculates guide token highlights for a player's fragment
func (ps *PuzzleService) CalculateGuideHighlights(grid *models.PuzzleGrid, fragmentID string,
	guideThreshold int) []models.Position {

	if grid == nil {
		return []models.Position{}
	}

	// Calculate reduction based on threshold
	reduction := guideThreshold * (grid.Size * grid.Size / 7)

	// Get highlights from grid
	return grid.GetGuideHighlights(fragmentID, reduction)
}

// ValidateFragmentMove validates if a fragment move is legal
func (ps *PuzzleService) ValidateFragmentMove(grid *models.PuzzleGrid, playerID string,
	fragmentID string, targetPos models.Position) error {

	if grid == nil {
		return fmt.Errorf("puzzle grid not initialized")
	}

	// Check if fragment exists
	fragment, exists := grid.Fragments[fragmentID]
	if !exists {
		return fmt.Errorf("fragment not found")
	}

	// Check ownership rules
	if fragment.IsOwned() && fragment.PlayerID != playerID {
		return fmt.Errorf("cannot move another player's fragment without permission")
	}

	// Check position bounds
	if targetPos.X < 0 || targetPos.X >= grid.Size ||
		targetPos.Y < 0 || targetPos.Y >= grid.Size {
		return fmt.Errorf("target position out of bounds")
	}

	return nil
}

// ExecuteRecommendedSwap executes a swap that was recommended and accepted
func (ps *PuzzleService) ExecuteRecommendedSwap(grid *models.PuzzleGrid,
	recommendationID string) error {

	ps.mu.Lock()
	rec, exists := ps.recommendations[recommendationID]
	ps.mu.Unlock()

	if !exists {
		return fmt.Errorf("recommendation not found")
	}

	if rec.Status != "accepted" {
		return fmt.Errorf("recommendation not accepted")
	}

	// Execute the swap
	err := grid.SwapFragments(rec.FromFragmentID, rec.ToFragmentID)
	if err != nil {
		return fmt.Errorf("failed to execute swap: %w", err)
	}

	// Update recommendation status
	ps.UpdateRecommendationStatus(recommendationID, "executed")

	return nil
}

// GetPuzzleImagePath returns the path to puzzle segment images
func (ps *PuzzleService) GetPuzzleImagePath(imageID string, gridSize int, segmentID string) string {
	return fmt.Sprintf("%s/%s/%dx%d/%s.png",
		config.PuzzleImagesPath, imageID, gridSize, gridSize, segmentID)
}

// GetAllSegmentPaths returns paths to all segments for a puzzle
func (ps *PuzzleService) GetAllSegmentPaths(imageID string, gridSize int) []string {
	segments := ps.generateSegmentIDs(gridSize)
	paths := make([]string, len(segments))

	for i, segment := range segments {
		paths[i] = ps.GetPuzzleImagePath(imageID, gridSize, segment)
	}

	return paths
}

// CalculatePreSolvedPieces determines which pieces should be pre-solved
func (ps *PuzzleService) CalculatePreSolvedPieces(anchorThreshold int) []int {
	totalPieces := constants.IndividualPuzzlePieces
	preSolvedCount := anchorThreshold * constants.PiecesPreSolvedPerThreshold

	if preSolvedCount > constants.MaxPreSolvedPieces {
		preSolvedCount = constants.MaxPreSolvedPieces
	}

	if preSolvedCount >= totalPieces {
		preSolvedCount = totalPieces - 4 // Always leave at least 4 pieces to solve
	}

	// Generate random indices for pre-solved pieces
	allIndices := make([]int, totalPieces)
	for i := range allIndices {
		allIndices[i] = i
	}

	// Shuffle and take first N indices
	rand.Shuffle(len(allIndices), func(i, j int) {
		allIndices[i], allIndices[j] = allIndices[j], allIndices[i]
	})

	preSolvedIndices := allIndices[:preSolvedCount]
	return preSolvedIndices
}

// StartPeriodicCleanup starts a goroutine to periodically clean up expired recommendations
func (ps *PuzzleService) StartPeriodicCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ps.CleanupExpiredRecommendations()
		}
	}()
}
