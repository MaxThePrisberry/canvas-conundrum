package test_helpers

import (
	"canvas-conundrum/config"
	"canvas-conundrum/models"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// TestFixtures provides test data
type TestFixtures struct {
	Players []*models.Player
	Host    *models.Host
	Game    *models.Game
	Trivia  []*models.TriviaQuestion
}

// NewTestFixtures creates test fixtures
func NewTestFixtures() *TestFixtures {
	return &TestFixtures{
		Players: createTestPlayers(),
		Host:    createTestHost(),
		Game:    createTestGameWithSettings(),
		Trivia:  createTestTriviaQuestions(),
	}
}

// createTestPlayers creates a set of test players
func createTestPlayers() []*models.Player {
	players := []*models.Player{
		createPlayerWithRole("player1", "Alice", models.RoleArtEnthusiast),
		createPlayerWithRole("player2", "Bob", models.RoleDetective),
		createPlayerWithRole("player3", "Charlie", models.RoleTourist),
		createPlayerWithRole("player4", "Diana", models.RoleJanitor),
	}

	// Set specialties
	players[0].Specialties = []models.TriviaCategory{models.CategoryGeneral}
	players[1].Specialties = []models.TriviaCategory{models.CategoryHistory}
	players[2].Specialties = []models.TriviaCategory{models.CategoryScience}
	players[3].Specialties = []models.TriviaCategory{models.CategoryMusic}

	return players
}

func createPlayerWithRole(id, name string, role models.Role) *models.Player {
	player := models.NewPlayer(id, nil)
	player.Name = name
	player.Role = role
	player.IsReady = true
	return player
}

// createTestHost creates a test host
func createTestHost() *models.Host {
	host := models.NewHost("test-host", nil)
	return host
}

// createTestGameWithSettings creates a game with specific settings
func createTestGameWithSettings() *models.Game {
	game := models.NewGame()
	game.Difficulty = models.DifficultyMedium
	game.PlayerCount = 4
	return game
}

// createTestTriviaQuestions creates sample trivia questions
func createTestTriviaQuestions() []*models.TriviaQuestion {
	questions := []*models.TriviaQuestion{}

	categories := []models.TriviaCategory{
		models.CategoryGeneral,
		models.CategoryGeography,
		models.CategoryHistory,
		models.CategoryMusic,
		models.CategoryScience,
		models.CategoryVideoGames,
	}

	difficulties := []models.TriviaDifficulty{
		models.DifficultyEasyTrivia,
		models.DifficultyMediumTrivia,
		models.DifficultyHardTrivia,
	}

	for i, cat := range categories {
		for j, diff := range difficulties {
			q := &models.TriviaQuestion{
				ID:            fmt.Sprintf("q-%s-%s-%d", cat, diff, i*3+j),
				Category:      cat,
				Difficulty:    diff,
				Question:      fmt.Sprintf("Test %s %s question?", cat, diff),
				Options:       []string{"Option A", "Option B", "Option C", "Option D"},
				CorrectAnswer: "Option A",
				CorrectIndex:  0,
			}
			questions = append(questions, q)
		}
	}

	return questions
}

// GameScenario represents a complete game scenario for testing
type GameScenario struct {
	Name              string
	PlayerCount       int
	Difficulty        models.DifficultyMode
	TokensEarned      map[models.TokenType]int
	ExpectedGridSize  int
	ExpectedPreSolved int
}

// GetGameScenarios returns various game scenarios for testing
func GetGameScenarios() []GameScenario {
	return []GameScenario{
		{
			Name:        "Small Easy Game",
			PlayerCount: 4,
			Difficulty:  models.DifficultyEasy,
			TokensEarned: map[models.TokenType]int{
				models.TokenAnchor:  10,
				models.TokenChronos: 10,
				models.TokenGuide:   10,
				models.TokenClarity: 10,
			},
			ExpectedGridSize:  3,
			ExpectedPreSolved: 2, // Threshold 1 for all tokens
		},
		{
			Name:        "Medium Game High Tokens",
			PlayerCount: 16,
			Difficulty:  models.DifficultyMedium,
			TokensEarned: map[models.TokenType]int{
				models.TokenAnchor:  30,
				models.TokenChronos: 30,
				models.TokenGuide:   30,
				models.TokenClarity: 30,
			},
			ExpectedGridSize:  5,
			ExpectedPreSolved: 7, // Threshold 2 for all tokens
		},
		{
			Name:        "Large Hard Game",
			PlayerCount: 64,
			Difficulty:  models.DifficultyHard,
			TokensEarned: map[models.TokenType]int{
				models.TokenAnchor:  50,
				models.TokenChronos: 50,
				models.TokenGuide:   50,
				models.TokenClarity: 50,
			},
			ExpectedGridSize:  8,
			ExpectedPreSolved: 11, // Threshold 3 for all tokens
		},
	}
}

// CreateMockTriviaFiles creates temporary trivia JSON files for testing
func CreateMockTriviaFiles() (string, func(), error) {
	// Create temp directory
	tempDir, err := ioutil.TempDir("", "trivia_test")
	if err != nil {
		return "", nil, err
	}

	// Create category directories and files
	categories := []string{"general", "geography", "history", "music", "science", "video_games"}
	difficulties := []string{"easy", "medium", "hard"}

	for _, cat := range categories {
		catDir := filepath.Join(tempDir, cat)
		if err := os.MkdirAll(catDir, 0755); err != nil {
			os.RemoveAll(tempDir)
			return "", nil, err
		}

		for _, diff := range difficulties {
			questions := createMockTriviaJSON(cat, diff)
			filePath := filepath.Join(catDir, diff+".json")
			if err := ioutil.WriteFile(filePath, questions, 0644); err != nil {
				os.RemoveAll(tempDir)
				return "", nil, err
			}
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup, nil
}

func createMockTriviaJSON(category, difficulty string) []byte {
	response := models.TriviaAPIResponse{
		ResponseCode: 0,
		Results: []models.RawTriviaQuestion{
			{
				Category:      category,
				Type:          "multiple",
				Difficulty:    difficulty,
				Question:      fmt.Sprintf("Test %s %s question 1?", category, difficulty),
				CorrectAnswer: "Correct Answer 1",
				Incorrect:     []string{"Wrong 1", "Wrong 2", "Wrong 3"},
			},
			{
				Category:      category,
				Type:          "multiple",
				Difficulty:    difficulty,
				Question:      fmt.Sprintf("Test %s %s question 2?", category, difficulty),
				CorrectAnswer: "Correct Answer 2",
				Incorrect:     []string{"Wrong A", "Wrong B", "Wrong C"},
			},
		},
	}

	data, _ := json.Marshal(response)
	return data
}

// Note: CreateTestGameManager and SetupTestGame removed to avoid import cycle
// These should be defined in the test files that need them

// ValidateWebSocketMessage validates a WebSocket message structure
func ValidateWebSocketMessage(data []byte) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Check required fields
	if _, ok := msg["event"]; !ok {
		return fmt.Errorf("missing 'event' field")
	}

	if _, ok := msg["payload"]; !ok {
		return fmt.Errorf("missing 'payload' field")
	}

	if _, ok := msg["timestamp"]; !ok {
		return fmt.Errorf("missing 'timestamp' field")
	}

	return nil
}

// AssertTokenThresholds validates token threshold calculations
func AssertTokenThresholds(tokens *models.TeamTokens, expectedThresholds map[models.TokenType]int) error {
	for tokenType, expected := range expectedThresholds {
		actual := tokens.GetThreshold(tokenType)
		if actual != expected {
			return fmt.Errorf("token %s: expected threshold %d, got %d", tokenType, expected, actual)
		}
	}
	return nil
}

// QRCodeHashes provides test QR code hashes matching the configured values
var QRCodeHashes = map[string]string{
	"anchor":  config.HashAnchorStation,
	"chronos": config.HashChronosStation,
	"guide":   config.HashGuideStation,
	"clarity": config.HashClarityStation,
}

// GetTestHostUUID returns the configured host UUID for testing
func GetTestHostUUID() string {
	return config.HostUUID
}
