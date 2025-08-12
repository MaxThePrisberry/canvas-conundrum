package models

import (
	"canvas-conundrum/constants"
	"time"
)

// GamePhase represents the current phase of the game
type GamePhase string

const (
	PhaseSetup             GamePhase = "setup"
	PhaseResourceGathering GamePhase = "resource_gathering"
	PhasePuzzleAssembly    GamePhase = "puzzle_assembly"
	PhaseAnalytics         GamePhase = "analytics"
)

// DifficultyMode represents the game difficulty
type DifficultyMode string

const (
	DifficultyEasy   DifficultyMode = "easy"
	DifficultyMedium DifficultyMode = "medium"
	DifficultyHard   DifficultyMode = "hard"
)

// TokenType represents the type of resource token
type TokenType string

const (
	TokenAnchor  TokenType = "anchor"
	TokenChronos TokenType = "chronos"
	TokenGuide   TokenType = "guide"
	TokenClarity TokenType = "clarity"
)

// TeamTokens holds the team's total tokens
type TeamTokens struct {
	AnchorTokens  int `json:"anchorTokens"`
	ChronosTokens int `json:"chronosTokens"`
	GuideTokens   int `json:"guideTokens"`
	ClarityTokens int `json:"clarityTokens"`
}

// GetTokenCount returns the count for a specific token type
func (tt *TeamTokens) GetTokenCount(tokenType TokenType) int {
	switch tokenType {
	case TokenAnchor:
		return tt.AnchorTokens
	case TokenChronos:
		return tt.ChronosTokens
	case TokenGuide:
		return tt.GuideTokens
	case TokenClarity:
		return tt.ClarityTokens
	default:
		return 0
	}
}

// AddTokens adds tokens of a specific type
func (tt *TeamTokens) AddTokens(tokenType TokenType, amount int) {
	switch tokenType {
	case TokenAnchor:
		tt.AnchorTokens += amount
	case TokenChronos:
		tt.ChronosTokens += amount
	case TokenGuide:
		tt.GuideTokens += amount
	case TokenClarity:
		tt.ClarityTokens += amount
	}
}

// GetThreshold returns the threshold level (0-3) for a token type
func (tt *TeamTokens) GetThreshold(tokenType TokenType) int {
	count := tt.GetTokenCount(tokenType)
	level := 0

	if count >= constants.TokenThreshold1 {
		level = 1
	}
	if count >= constants.TokenThreshold2 {
		level = 2
	}
	if count >= constants.TokenThreshold3 {
		level = 3
	}
	return level
}

// GetTotal returns the total number of all tokens
func (tt *TeamTokens) GetTotal() int {
	return tt.AnchorTokens + tt.ChronosTokens + tt.GuideTokens + tt.ClarityTokens
}

// NewTeamTokens creates a new TeamTokens instance
func NewTeamTokens() *TeamTokens {
	return &TeamTokens{
		AnchorTokens:  0,
		ChronosTokens: 0,
		GuideTokens:   0,
		ClarityTokens: 0,
	}
}

// Game represents the main game state
type Game struct {
	ID             string         `json:"id"`
	CurrentPhase   GamePhase      `json:"currentPhase"`
	Difficulty     DifficultyMode `json:"difficulty"`
	GameStarted    bool           `json:"gameStarted"`
	StartTime      time.Time      `json:"startTime"`
	PhaseStartTime time.Time      `json:"phaseStartTime"`
	PlayerCount    int            `json:"playerCount"`

	// Resource gathering phase
	CurrentRound   int         `json:"currentRound"`
	RoundStartTime time.Time   `json:"roundStartTime"`
	TeamTokens     *TeamTokens `json:"teamTokens"`

	// Puzzle phase
	PuzzleGrid         *PuzzleGrid `json:"puzzleGrid"`
	PuzzleStartTime    time.Time   `json:"puzzleStartTime"`
	PuzzleTimerStarted bool        `json:"puzzleTimerStarted"`
	PuzzleCompleted    bool        `json:"puzzleCompleted"`
	PuzzleSuccess      bool        `json:"puzzleSuccess"`

	// Game configuration
	MinPlayers int    `json:"minPlayers"`
	MaxPlayers int    `json:"maxPlayers"`
	ImageID    string `json:"imageId"`

	// Analytics
	TotalQuestionsDelivered int     `json:"totalQuestionsDelivered"`
	TotalCorrectAnswers     int     `json:"totalCorrectAnswers"`
	CompletionTime          float64 `json:"completionTime"`
	TeamScore               int     `json:"teamScore"`
}

// NewGame creates a new game instance
func NewGame() *Game {
	return &Game{
		ID:           generateGameID(),
		CurrentPhase: PhaseSetup,
		Difficulty:   DifficultyMedium,
		GameStarted:  false,
		MinPlayers:   constants.MinPlayers,
		MaxPlayers:   constants.MaxPlayers,
		TeamTokens:   NewTeamTokens(),
		ImageID:      "nature_image", // Default puzzle image
		PlayerCount:  0,
	}
}

// generateGameID creates a unique game ID
func generateGameID() string {
	return "game-" + time.Now().Format("20060102-150405")
}

// NextPhase transitions to the next game phase
func (g *Game) NextPhase() GamePhase {
	switch g.CurrentPhase {
	case PhaseSetup:
		g.CurrentPhase = PhaseResourceGathering
	case PhaseResourceGathering:
		g.CurrentPhase = PhasePuzzleAssembly
	case PhasePuzzleAssembly:
		g.CurrentPhase = PhaseAnalytics
	}
	g.PhaseStartTime = time.Now()
	return g.CurrentPhase
}

// SetDifficulty sets the game difficulty
func (g *Game) SetDifficulty(difficulty DifficultyMode) {
	g.Difficulty = difficulty
}

// GetGridSize returns the puzzle grid size based on player count
func (g *Game) GetGridSize() int {
	if g.PlayerCount <= 8 {
		return 3
	} else if g.PlayerCount <= 16 {
		return 4
	} else if g.PlayerCount <= 24 {
		return 5
	} else if g.PlayerCount <= 36 {
		return 6
	} else if g.PlayerCount <= 48 {
		return 7
	}
	return 8
}

// StartResourceGathering transitions to resource gathering phase
func (g *Game) StartResourceGathering() {
	g.CurrentPhase = PhaseResourceGathering
	g.PhaseStartTime = time.Now()
	g.GameStarted = true
	g.StartTime = time.Now()
	g.CurrentRound = 1 // Start at round 1
}

// StartNextRound advances to the next resource gathering round
func (g *Game) StartNextRound() {
	g.CurrentRound++
	g.RoundStartTime = time.Now()
}

// StartPuzzlePhase transitions to puzzle assembly phase
func (g *Game) StartPuzzlePhase(playerCount int) {
	g.CurrentPhase = PhasePuzzleAssembly
	g.PhaseStartTime = time.Now()

	// Initialize puzzle grid based on player count
	gridSize := constants.GetGridSizeForPlayerCount(playerCount)
	g.PuzzleGrid = NewPuzzleGrid(gridSize)
}

// StartPuzzleTimer starts the puzzle timer
func (g *Game) StartPuzzleTimer() {
	g.PuzzleTimerStarted = true
	g.PuzzleStartTime = time.Now()
}

// GetPuzzleTimeRemaining returns seconds remaining in puzzle phase
func (g *Game) GetPuzzleTimeRemaining() int {
	if !g.PuzzleTimerStarted {
		return g.GetTotalPuzzleTime()
	}

	elapsed := time.Since(g.PuzzleStartTime).Seconds()
	total := g.GetTotalPuzzleTime()
	remaining := total - int(elapsed)

	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// GetTotalPuzzleTime returns total puzzle time including bonuses
func (g *Game) GetTotalPuzzleTime() int {
	baseTime := constants.PuzzleBaseTime
	chronosBonus := g.TeamTokens.GetThreshold(TokenChronos) * constants.TimeExtensionPerThreshold

	// Apply difficulty modifier
	var modifier float64
	switch g.Difficulty {
	case DifficultyEasy:
		modifier = constants.EasyTimeMultiplier
	case DifficultyHard:
		modifier = constants.HardTimeMultiplier
	default:
		modifier = constants.MediumTimeMultiplier
	}

	return int(float64(baseTime+chronosBonus) * modifier)
}

// GetPreSolvedPieces returns number of pieces to pre-solve based on anchor tokens
func (g *Game) GetPreSolvedPieces() int {
	threshold := g.TeamTokens.GetThreshold(TokenAnchor)
	pieces := threshold * constants.PiecesPreSolvedPerThreshold
	if pieces > constants.MaxPreSolvedPieces {
		pieces = constants.MaxPreSolvedPieces
	}
	return pieces
}

// GetClarityPreviewTime returns clarity preview duration in seconds
func (g *Game) GetClarityPreviewTime() int {
	baseTime := constants.ClarityBasePreviewTime
	threshold := g.TeamTokens.GetThreshold(TokenClarity)
	return baseTime + (threshold * constants.PreviewTimePerThreshold)
}

// GetGuideHighlightReduction returns number of squares to remove per threshold
func (g *Game) GetGuideHighlightReduction() int {
	if g.PuzzleGrid == nil {
		return 0
	}
	gridSquares := g.PuzzleGrid.Size * g.PuzzleGrid.Size
	return gridSquares / 7
}

// CompleteGame marks the game as complete
func (g *Game) CompleteGame(success bool) {
	g.CurrentPhase = PhaseAnalytics
	g.PuzzleCompleted = true
	g.PuzzleSuccess = success
	if success {
		g.CompletionTime = time.Since(g.PuzzleStartTime).Seconds()
	}
}

// Reset resets the game to initial state
func (g *Game) Reset() {
	*g = *NewGame()
}
