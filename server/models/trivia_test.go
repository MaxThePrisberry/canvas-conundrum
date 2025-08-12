package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawTriviaQuestionToTriviaQuestion(t *testing.T) {
	raw := &RawTriviaQuestion{
		Category:      "General Knowledge",
		Type:          "multiple",
		Difficulty:    "easy",
		Question:      "What is 2 &plus; 2?",
		CorrectAnswer: "4",
		Incorrect:     []string{"3", "5", "6"},
	}

	question := raw.ToTriviaQuestion()

	assert.NotNil(t, question)
	assert.NotEmpty(t, question.ID)
	assert.Equal(t, CategoryGeneral, question.Category)
	assert.Equal(t, DifficultyEasyTrivia, question.Difficulty)
	assert.Equal(t, "What is 2 + 2?", question.Question) // HTML decoded
	assert.Equal(t, "4", question.CorrectAnswer)
	assert.Len(t, question.Options, 4)
	assert.Contains(t, question.Options, "4")
	assert.Contains(t, question.Options, "3")
	assert.Contains(t, question.Options, "5")
	assert.Contains(t, question.Options, "6")
	assert.True(t, question.CorrectIndex >= 0 && question.CorrectIndex < 4)
	assert.Equal(t, "4", question.Options[question.CorrectIndex])
}

func TestHTMLDecoding(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"&amp;", "&"},
		{"&lt;", "<"},
		{"&gt;", ">"},
		{"&quot;", "\""},
		{"&#039;", "'"},
		{"&plus;", "+"},
		{"&minus;", "−"}, // Unicode minus sign U+2212
		{"What&#039;s this?", "What's this?"},
	}

	for _, tt := range tests {
		raw := &RawTriviaQuestion{
			Category:      "Test",
			Type:          "multiple",
			Difficulty:    "easy",
			Question:      tt.input,
			CorrectAnswer: "Test",
			Incorrect:     []string{"A", "B", "C"},
		}

		question := raw.ToTriviaQuestion()
		assert.Equal(t, tt.expected, question.Question)
	}
}

func TestCategoryFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected TriviaCategory
	}{
		{"general", CategoryGeneral},
		{"General", CategoryGeneral},
		{"general_knowledge", CategoryGeneral},
		{"General Knowledge", CategoryGeneral},
		{"geography", CategoryGeography},
		{"Geography", CategoryGeography},
		{"history", CategoryHistory},
		{"History", CategoryHistory},
		{"music", CategoryMusic},
		{"Music", CategoryMusic},
		{"entertainment:_music", CategoryMusic},
		{"science", CategoryScience},
		{"Science", CategoryScience},
		{"science_&_nature", CategoryScience},
		{"video_games", CategoryVideoGames},
		{"Video Games", CategoryVideoGames},
		{"entertainment:_video_games", CategoryVideoGames},
		{"unknown", CategoryGeneral}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := categoryFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDifficultyFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected TriviaDifficulty
	}{
		{"easy", DifficultyEasyTrivia},
		{"Easy", DifficultyEasyTrivia},
		{"medium", DifficultyMediumTrivia},
		{"Medium", DifficultyMediumTrivia},
		{"hard", DifficultyHardTrivia},
		{"Hard", DifficultyHardTrivia},
		{"unknown", DifficultyMediumTrivia}, // Default
		{"", DifficultyMediumTrivia},        // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := difficultyFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuestionPool(t *testing.T) {
	t.Run("NewQuestionPool", func(t *testing.T) {
		pool := NewQuestionPool()

		assert.NotNil(t, pool)
		assert.Empty(t, pool.Questions)
		assert.Empty(t, pool.Used)
		assert.Equal(t, 0, pool.Index)
	})

	t.Run("AddQuestion", func(t *testing.T) {
		pool := NewQuestionPool()

		q1 := &TriviaQuestion{ID: "q1", Question: "Question 1"}
		q2 := &TriviaQuestion{ID: "q2", Question: "Question 2"}

		pool.AddQuestion(q1)
		pool.AddQuestion(q2)

		assert.Len(t, pool.Questions, 2)
		assert.Contains(t, pool.Questions, q1)
		assert.Contains(t, pool.Questions, q2)
	})

	t.Run("GetNextQuestion", func(t *testing.T) {
		pool := NewQuestionPool()

		// Empty pool returns nil
		assert.Nil(t, pool.GetNextQuestion())

		// Add questions
		questions := []*TriviaQuestion{
			{ID: "q1", Question: "Question 1"},
			{ID: "q2", Question: "Question 2"},
			{ID: "q3", Question: "Question 3"},
		}

		for _, q := range questions {
			pool.AddQuestion(q)
		}

		// Get questions
		used := make(map[string]bool)
		for i := 0; i < 3; i++ {
			q := pool.GetNextQuestion()
			assert.NotNil(t, q)
			assert.False(t, used[q.ID], "Question %s already used", q.ID)
			used[q.ID] = true
		}

		// All questions used, should reset and shuffle
		q := pool.GetNextQuestion()
		assert.NotNil(t, q)
		assert.Contains(t, []string{"q1", "q2", "q3"}, q.ID)
	})

	t.Run("QuestionPoolReset", func(t *testing.T) {
		pool := NewQuestionPool()

		// Add questions
		for i := 0; i < 5; i++ {
			pool.AddQuestion(&TriviaQuestion{
				ID:       string(rune('a' + i)),
				Question: string(rune('a' + i)),
			})
		}

		// Use all questions
		for i := 0; i < 5; i++ {
			q := pool.GetNextQuestion()
			assert.NotNil(t, q)
		}

		// Next call should reset
		assert.Equal(t, 5, len(pool.Used))
		q := pool.GetNextQuestion()
		assert.NotNil(t, q)
		// After reset, only the newly returned question should be marked as used
		assert.Equal(t, 1, len(pool.Used))
	})
}

func TestTriviaAnswer(t *testing.T) {
	answer := &TriviaAnswer{
		PlayerID:       "player1",
		QuestionID:     "question1",
		SelectedAnswer: "Option A",
		AnswerIndex:    0,
		TimeElapsed:    5.5,
		Correct:        true,
		TokensEarned:   3,
		Timestamp:      time.Now(),
	}

	assert.Equal(t, "player1", answer.PlayerID)
	assert.Equal(t, "question1", answer.QuestionID)
	assert.Equal(t, "Option A", answer.SelectedAnswer)
	assert.Equal(t, 0, answer.AnswerIndex)
	assert.Equal(t, 5.5, answer.TimeElapsed)
	assert.True(t, answer.Correct)
	assert.Equal(t, 3, answer.TokensEarned)
	assert.NotZero(t, answer.Timestamp)
}

func TestLoadQuestionsFromJSON(t *testing.T) {
	t.Run("Valid JSON", func(t *testing.T) {
		jsonData := `{
			"response_code": 0,
			"results": [
				{
					"category": "General Knowledge",
					"type": "multiple",
					"difficulty": "easy",
					"question": "What is 2+2?",
					"correct_answer": "4",
					"incorrect_answers": ["3", "5", "6"]
				},
				{
					"category": "Science",
					"type": "multiple",
					"difficulty": "medium",
					"question": "What is H2O?",
					"correct_answer": "Water",
					"incorrect_answers": ["Oxygen", "Hydrogen", "Carbon"]
				}
			]
		}`

		questions, err := LoadQuestionsFromJSON([]byte(jsonData))

		require.NoError(t, err)
		assert.Len(t, questions, 2)

		assert.Equal(t, CategoryGeneral, questions[0].Category)
		assert.Equal(t, DifficultyEasyTrivia, questions[0].Difficulty)
		assert.Equal(t, "What is 2+2?", questions[0].Question)
		assert.Equal(t, "4", questions[0].CorrectAnswer)

		assert.Equal(t, CategoryScience, questions[1].Category)
		assert.Equal(t, DifficultyMediumTrivia, questions[1].Difficulty)
		assert.Equal(t, "What is H2O?", questions[1].Question)
		assert.Equal(t, "Water", questions[1].CorrectAnswer)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		jsonData := `{invalid json}`

		questions, err := LoadQuestionsFromJSON([]byte(jsonData))

		assert.Error(t, err)
		assert.Nil(t, questions)
	})

	t.Run("Empty Results", func(t *testing.T) {
		jsonData := `{
			"response_code": 0,
			"results": []
		}`

		questions, err := LoadQuestionsFromJSON([]byte(jsonData))

		require.NoError(t, err)
		assert.Empty(t, questions)
	})
}

func TestTriviaAPIResponse(t *testing.T) {
	response := TriviaAPIResponse{
		ResponseCode: 0,
		Results: []RawTriviaQuestion{
			{
				Category:      "Test",
				Type:          "multiple",
				Difficulty:    "easy",
				Question:      "Test?",
				CorrectAnswer: "Yes",
				Incorrect:     []string{"No", "Maybe", "Unknown"},
			},
		},
	}

	// Should be able to marshal/unmarshal
	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded TriviaAPIResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, response.ResponseCode, decoded.ResponseCode)
	assert.Len(t, decoded.Results, 1)
	assert.Equal(t, response.Results[0].Question, decoded.Results[0].Question)
}

func TestTriviaQuestionSpecialtyFields(t *testing.T) {
	question := &TriviaQuestion{
		ID:             "test",
		Category:       CategoryScience,
		Difficulty:     DifficultyHardTrivia,
		Question:       "Complex science question?",
		Options:        []string{"A", "B", "C", "D"},
		CorrectAnswer:  "A",
		CorrectIndex:   0,
		IsSpecialty:    true,
		SpecialtyBonus: true,
	}

	assert.True(t, question.IsSpecialty)
	assert.True(t, question.SpecialtyBonus)

	// Test JSON serialization preserves fields
	data, err := json.Marshal(question)
	require.NoError(t, err)

	var decoded TriviaQuestion
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, question.IsSpecialty, decoded.IsSpecialty)
	assert.Equal(t, question.SpecialtyBonus, decoded.SpecialtyBonus)
}

func TestOptionsShuffling(t *testing.T) {
	// Test that options are shuffled and correct answer index is updated
	raw := &RawTriviaQuestion{
		Category:      "Test",
		Type:          "multiple",
		Difficulty:    "easy",
		Question:      "Test?",
		CorrectAnswer: "Correct",
		Incorrect:     []string{"Wrong1", "Wrong2", "Wrong3"},
	}

	// Convert multiple times to check shuffling
	shufflePatterns := make(map[string]int)
	for i := 0; i < 10; i++ {
		question := raw.ToTriviaQuestion()
		pattern := ""
		for _, opt := range question.Options {
			pattern += opt[0:1] // First letter of each option
		}
		shufflePatterns[pattern]++

		// Verify correct answer index is always accurate
		assert.Equal(t, "Correct", question.Options[question.CorrectIndex])
	}

	// Should have different shuffle patterns (not always the same)
	// Note: There's a small chance this could fail randomly
	assert.Greater(t, len(shufflePatterns), 1, "Options should be shuffled differently")
}
