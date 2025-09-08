package services

import (
	"canvas-conundrum/models"
	"canvas-conundrum/test_helpers"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTriviaService(t *testing.T) {
	service := NewTriviaService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.pools)
	assert.Equal(t, "./trivia", service.basePath)
}

func TestTriviaServiceGetPoolKey(t *testing.T) {
	service := NewTriviaService()

	tests := []struct {
		category   models.TriviaCategory
		difficulty models.TriviaDifficulty
		expected   string
	}{
		{models.CategoryScience, models.DifficultyEasyTrivia, "science_easy"},
		{models.CategoryHistory, models.DifficultyMediumTrivia, "history_medium"},
		{models.CategoryGeography, models.DifficultyHardTrivia, "geography_hard"},
	}

	for _, tt := range tests {
		result := service.getPoolKey(tt.category, tt.difficulty)
		assert.Equal(t, tt.expected, result)
	}
}

func TestTriviaServiceLoadQuestions(t *testing.T) {
	// Create temporary directory structure for test files
	tempDir, err := ioutil.TempDir("", "trivia_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	service := NewTriviaService()
	service.basePath = tempDir

	t.Run("Load Valid Questions", func(t *testing.T) {
		// Create test directory structure
		scienceDir := filepath.Join(tempDir, "science")
		err := os.MkdirAll(scienceDir, 0755)
		require.NoError(t, err)

		// Create test JSON file with valid questions
		testQuestions := `{
			"results": [
				{
					"question": "What is the chemical symbol for gold?",
					"correct_answer": "Au",
					"incorrect_answers": ["Ag", "Fe", "Cu"],
					"category": "Science",
					"type": "multiple",
					"difficulty": "easy"
				},
				{
					"question": "What planet is known as the Red Planet?",
					"correct_answer": "Mars",
					"incorrect_answers": ["Venus", "Jupiter", "Saturn"],
					"category": "Science",
					"type": "multiple",
					"difficulty": "easy"
				}
			]
		}`

		easyFile := filepath.Join(scienceDir, "easy.json")
		err = ioutil.WriteFile(easyFile, []byte(testQuestions), 0644)
		require.NoError(t, err)

		// Load questions
		err = service.LoadQuestions()

		// Should not error even if most files don't exist
		assert.NoError(t, err)

		// Check that the questions were loaded
		poolKey := service.getPoolKey(models.CategoryScience, models.DifficultyEasyTrivia)
		pool, exists := service.pools[poolKey]
		require.True(t, exists)
		assert.NotNil(t, pool)
	})

	t.Run("Load From Non-existent Directory", func(t *testing.T) {
		service := NewTriviaService()
		service.basePath = "/nonexistent/path"

		// Should not panic when files don't exist
		err := service.LoadQuestions()
		assert.NoError(t, err) // Service logs warnings but doesn't error
	})

	t.Run("Load Invalid JSON", func(t *testing.T) {
		// Create directory with invalid JSON
		invalidDir := filepath.Join(tempDir, "invalid")
		err := os.MkdirAll(invalidDir, 0755)
		require.NoError(t, err)

		invalidFile := filepath.Join(invalidDir, "easy.json")
		err = ioutil.WriteFile(invalidFile, []byte("invalid json"), 0644)
		require.NoError(t, err)

		service := NewTriviaService()
		service.basePath = tempDir

		// Should handle invalid JSON gracefully
		err = service.LoadQuestions()
		assert.NoError(t, err) // Logs warning but doesn't error
	})
}

func TestTriviaServiceGetQuestionsForRound(t *testing.T) {
	service := NewTriviaService()

	// Create mock question pools
	service.pools["science_easy"] = models.NewQuestionPool()
	service.pools["science_easy"].AddQuestion(&models.TriviaQuestion{
		ID:            "q1",
		Question:      "Test question 1",
		CorrectAnswer: "Answer 1",
		Options:       []string{"Answer 1", "Wrong 1", "Wrong 2", "Wrong 3"},
		CorrectIndex:  0,
		Category:      models.CategoryScience,
		Difficulty:    models.DifficultyEasyTrivia,
	})

	service.pools["history_medium"] = models.NewQuestionPool()
	service.pools["history_medium"].AddQuestion(&models.TriviaQuestion{
		ID:            "q2",
		Question:      "Test question 2",
		CorrectAnswer: "Answer 2",
		Options:       []string{"Answer 2", "Wrong 1", "Wrong 2", "Wrong 3"},
		CorrectIndex:  0,
		Category:      models.CategoryHistory,
		Difficulty:    models.DifficultyMediumTrivia,
	})

	t.Run("No Players", func(t *testing.T) {
		players := make(map[string]*models.Player)
		questions := service.GetQuestionsForRound(players)

		assert.Empty(t, questions)
	})

	t.Run("Players With Specialties", func(t *testing.T) {
		players := map[string]*models.Player{
			"player1": {
				ID:          "player1",
				IsActive:    true,
				Specialties: []models.TriviaCategory{models.CategoryScience},
			},
			"player2": {
				ID:          "player2",
				IsActive:    true,
				Specialties: []models.TriviaCategory{models.CategoryHistory},
			},
		}

		questions := service.GetQuestionsForRound(players)

		assert.Len(t, questions, 2)
		assert.Contains(t, questions, "player1")
		assert.Contains(t, questions, "player2")
	})

	t.Run("Inactive Players", func(t *testing.T) {
		players := map[string]*models.Player{
			"player1": {
				ID:          "player1",
				IsActive:    false, // Inactive
				Specialties: []models.TriviaCategory{models.CategoryScience},
			},
			"player2": {
				ID:          "player2",
				IsActive:    true,
				Specialties: []models.TriviaCategory{models.CategoryHistory},
			},
		}

		questions := service.GetQuestionsForRound(players)

		assert.Len(t, questions, 1)
		assert.Contains(t, questions, "player2")
		assert.NotContains(t, questions, "player1")
	})

	t.Run("Players Without Specialties", func(t *testing.T) {
		players := map[string]*models.Player{
			"player1": {
				ID:          "player1",
				IsActive:    true,
				Specialties: []models.TriviaCategory{}, // No specialties
			},
		}

		questions := service.GetQuestionsForRound(players)

		// Should still get a question (general category or random)
		assert.Len(t, questions, 1)
		assert.Contains(t, questions, "player1")
	})
}

// GetQuestionForPlayer method doesn't exist in actual implementation
// The trivia service uses GetQuestionsForRound to get questions for all players
// Removing this test as the method doesn't exist in the actual service

func TestTriviaServiceProcessAnswer(t *testing.T) {
	service := NewTriviaService()
	resetGameManager()
	gameManager := GetGameInstance()

	// Create test player and add to game
	player := &models.Player{
		ID:             "player1",
		Name:           "Test Player",
		CurrentStation: "anchor",
		Role:           models.RoleArtEnthusiast,
	}
	gameManager.AddPlayer(player)

	// Create and store a test question
	question := &models.TriviaQuestion{
		ID:            "q1",
		Question:      "Test question",
		CorrectAnswer: "Correct Answer",
		Options:       []string{"Correct Answer", "Wrong1", "Wrong2", "Wrong3"},
		CorrectIndex:  0,
		Category:      models.CategoryScience,
		Difficulty:    models.DifficultyEasyTrivia,
	}

	// Add question to a pool so GetQuestionByID can find it
	pool := models.NewQuestionPool()
	pool.AddQuestion(question)
	service.pools["science_easy"] = pool

	t.Run("Correct Answer", func(t *testing.T) {
		answer, tokens, tokenType := service.ProcessAnswer("player1", "q1", "Correct Answer", 0, 2.5)

		assert.NotNil(t, answer)
		assert.True(t, answer.Correct)
		assert.Equal(t, "Correct Answer", answer.SelectedAnswer)
		assert.Greater(t, tokens, 0)
		assert.NotEmpty(t, tokenType)
	})

	t.Run("Incorrect Answer", func(t *testing.T) {
		answer, tokens, tokenType := service.ProcessAnswer("player1", "q1", "Wrong Answer", 1, 3.0)

		assert.NotNil(t, answer)
		assert.False(t, answer.Correct)
		assert.Equal(t, "Wrong Answer", answer.SelectedAnswer)
		assert.Equal(t, 0, tokens)
		assert.Empty(t, tokenType)
	})

	t.Run("Invalid Player", func(t *testing.T) {
		answer, tokens, tokenType := service.ProcessAnswer("invalid", "q1", "Correct Answer", 0, 2.5)

		assert.Nil(t, answer)
		assert.Equal(t, 0, tokens)
		assert.Empty(t, tokenType)
	})

	t.Run("Invalid Question", func(t *testing.T) {
		answer, tokens, tokenType := service.ProcessAnswer("player1", "invalid", "Correct Answer", 0, 2.5)

		assert.Nil(t, answer)
		assert.Equal(t, 0, tokens)
		assert.Empty(t, tokenType)
	})
}

func TestTriviaServiceConcurrency(t *testing.T) {
	service := NewTriviaService()
	resetGameManager()
	gameManager := GetGameInstance()

	// Add some test data
	pool := models.NewQuestionPool()
	pool.AddQuestion(&models.TriviaQuestion{
		ID:            "q1",
		Question:      "Test question",
		CorrectAnswer: "Answer",
		Options:       []string{"Answer", "W1", "W2", "W3"},
		CorrectIndex:  0,
		Category:      models.CategoryScience,
		Difficulty:    models.DifficultyEasyTrivia,
	})
	service.pools["science_easy"] = pool

	// Add players to test concurrency
	players := make(map[string]*models.Player)
	for i := 0; i < 3; i++ {
		playerID := "player" + string(rune(i+'1'))
		player := test_helpers.CreateTestPlayer(playerID)
		player.IsActive = true
		player.Specialties = []models.TriviaCategory{models.CategoryScience}
		players[playerID] = player
		gameManager.AddPlayer(player)
	}

	// Test concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.GetQuestionsForRound(players)
			_ = service.GetQuestionByID("q1")
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Test should complete without race conditions
	assert.True(t, true)
}

func TestTriviaServiceGetQuestionByID(t *testing.T) {
	service := NewTriviaService()

	// Add test question to pool
	question := &models.TriviaQuestion{
		ID:            "test-question-id",
		Question:      "Test question",
		CorrectAnswer: "Test answer",
		Options:       []string{"Test answer", "Wrong1", "Wrong2", "Wrong3"},
		CorrectIndex:  0,
		Category:      models.CategoryScience,
		Difficulty:    models.DifficultyEasyTrivia,
	}

	pool := models.NewQuestionPool()
	pool.AddQuestion(question)
	service.pools["science_easy"] = pool

	t.Run("Existing Question", func(t *testing.T) {
		result := service.GetQuestionByID("test-question-id")
		assert.NotNil(t, result)
		assert.Equal(t, "test-question-id", result.ID)
		assert.Equal(t, "Test question", result.Question)
	})

	t.Run("Non-existent Question", func(t *testing.T) {
		result := service.GetQuestionByID("non-existent")
		assert.Nil(t, result)
	})
}

func TestTriviaServiceIntegration(t *testing.T) {
	service := NewTriviaService()
	resetGameManager()
	gameManager := GetGameInstance()

	// Create test players
	player1 := test_helpers.CreateTestPlayer("player1")
	player1.IsActive = true
	player1.Specialties = []models.TriviaCategory{models.CategoryScience}
	player1.CurrentStation = "anchor"

	player2 := test_helpers.CreateTestPlayer("player2")
	player2.IsActive = true
	player2.Specialties = []models.TriviaCategory{models.CategoryHistory}
	player2.CurrentStation = "chronos"

	players := map[string]*models.Player{
		"player1": player1,
		"player2": player2,
	}

	// Add players to game manager
	gameManager.AddPlayer(player1)
	gameManager.AddPlayer(player2)

	// Add mock questions
	pool1 := models.NewQuestionPool()
	pool1.AddQuestion(&models.TriviaQuestion{
		ID:            "q1",
		Question:      "Science question",
		CorrectAnswer: "Science answer",
		Options:       []string{"Science answer", "W1", "W2", "W3"},
		CorrectIndex:  0,
		Category:      models.CategoryScience,
		Difficulty:    models.DifficultyEasyTrivia,
	})
	service.pools["science_easy"] = pool1

	pool2 := models.NewQuestionPool()
	pool2.AddQuestion(&models.TriviaQuestion{
		ID:            "q2",
		Question:      "History question",
		CorrectAnswer: "History answer",
		Options:       []string{"History answer", "W1", "W2", "W3"},
		CorrectIndex:  0,
		Category:      models.CategoryHistory,
		Difficulty:    models.DifficultyEasyTrivia,
	})
	service.pools["history_easy"] = pool2

	t.Run("Full Round Flow", func(t *testing.T) {
		// Get questions for round
		questions := service.GetQuestionsForRound(players)

		// Should have questions for both players
		assert.Len(t, questions, 2)

		// Process answers for each player
		for playerID, question := range questions {
			if question != nil {
				// Test correct answer
				answer, tokens, tokenType := service.ProcessAnswer(playerID, question.ID, question.CorrectAnswer, 0, 2.5)
				assert.NotNil(t, answer)
				assert.True(t, answer.Correct)
				assert.Greater(t, tokens, 0)
				assert.NotEmpty(t, tokenType)

				// Test incorrect answer
				answer, tokens, tokenType = service.ProcessAnswer(playerID, question.ID, "Wrong", 1, 3.0)
				assert.NotNil(t, answer)
				assert.False(t, answer.Correct)
				assert.Equal(t, 0, tokens)
				assert.Empty(t, tokenType)
			}
		}
	})
}
