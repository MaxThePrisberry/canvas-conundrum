package models

import (
	"encoding/json"
	"html"
	"math/rand"
	"strings"
	"time"
)

// TriviaCategory represents a trivia category
type TriviaCategory string

const (
	CategoryGeneral    TriviaCategory = "general"
	CategoryGeography  TriviaCategory = "geography"
	CategoryHistory    TriviaCategory = "history"
	CategoryMusic      TriviaCategory = "music"
	CategoryScience    TriviaCategory = "science"
	CategoryVideoGames TriviaCategory = "video_games"
)

// TriviaDifficulty represents question difficulty
type TriviaDifficulty string

const (
	DifficultyEasyTrivia   TriviaDifficulty = "easy"
	DifficultyMediumTrivia TriviaDifficulty = "medium"
	DifficultyHardTrivia   TriviaDifficulty = "hard"
)

// TriviaQuestion represents a trivia question
type TriviaQuestion struct {
	ID             string           `json:"id"`
	Category       TriviaCategory   `json:"category"`
	Difficulty     TriviaDifficulty `json:"difficulty"`
	Question       string           `json:"question"`
	Options        []string         `json:"options"`
	CorrectAnswer  string           `json:"correctAnswer"`
	CorrectIndex   int              `json:"correctIndex"`
	IsSpecialty    bool             `json:"isSpecialty,omitempty"`
	SpecialtyBonus bool             `json:"specialtyBonus,omitempty"`
}

// RawTriviaQuestion represents the raw JSON structure from trivia files
type RawTriviaQuestion struct {
	Category      string   `json:"category"`
	Type          string   `json:"type"`
	Difficulty    string   `json:"difficulty"`
	Question      string   `json:"question"`
	CorrectAnswer string   `json:"correct_answer"`
	Incorrect     []string `json:"incorrect_answers"`
}

// ToTriviaQuestion converts raw question to game question format
func (rq *RawTriviaQuestion) ToTriviaQuestion() *TriviaQuestion {
	// Decode HTML entities
	question := html.UnescapeString(rq.Question)
	correctAnswer := html.UnescapeString(rq.CorrectAnswer)

	// Create options array with correct and incorrect answers
	options := make([]string, 0, len(rq.Incorrect)+1)
	for _, incorrect := range rq.Incorrect {
		options = append(options, html.UnescapeString(incorrect))
	}
	options = append(options, correctAnswer)

	// Shuffle options
	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	// Find correct answer index
	correctIndex := -1
	for i, opt := range options {
		if opt == correctAnswer {
			correctIndex = i
			break
		}
	}

	// Generate unique ID
	id := generateQuestionID(rq.Category, rq.Difficulty)

	return &TriviaQuestion{
		ID:            id,
		Category:      categoryFromString(rq.Category),
		Difficulty:    difficultyFromString(rq.Difficulty),
		Question:      question,
		Options:       options,
		CorrectAnswer: correctAnswer,
		CorrectIndex:  correctIndex,
	}
}

// generateQuestionID creates a unique question ID
func generateQuestionID(category, difficulty string) string {
	timestamp := time.Now().UnixNano()
	random := rand.Intn(10000)
	return strings.ToLower(category) + "_" + difficulty + "_" +
		string(rune(timestamp%1000)) + "_" + string(rune(random))
}

// categoryFromString converts string to TriviaCategory
func categoryFromString(s string) TriviaCategory {
	s = strings.ToLower(strings.ReplaceAll(s, " ", "_"))
	switch s {
	case "general", "general_knowledge":
		return CategoryGeneral
	case "geography":
		return CategoryGeography
	case "history":
		return CategoryHistory
	case "music", "entertainment:_music":
		return CategoryMusic
	case "science", "science_&_nature":
		return CategoryScience
	case "video_games", "entertainment:_video_games":
		return CategoryVideoGames
	default:
		return CategoryGeneral
	}
}

// difficultyFromString converts string to TriviaDifficulty
func difficultyFromString(s string) TriviaDifficulty {
	switch strings.ToLower(s) {
	case "easy":
		return DifficultyEasyTrivia
	case "hard":
		return DifficultyHardTrivia
	default:
		return DifficultyMediumTrivia
	}
}

// TriviaAnswer represents a player's answer to a trivia question
type TriviaAnswer struct {
	PlayerID       string    `json:"playerId"`
	QuestionID     string    `json:"questionId"`
	SelectedAnswer string    `json:"selectedAnswer"`
	AnswerIndex    int       `json:"answerIndex"`
	TimeElapsed    float64   `json:"timeElapsed"`
	Correct        bool      `json:"correct"`
	TokensEarned   int       `json:"tokensEarned"`
	Timestamp      time.Time `json:"timestamp"`
}

// QuestionPool manages a pool of trivia questions
type QuestionPool struct {
	Questions []*TriviaQuestion
	Used      map[string]bool
	Index     int
}

// NewQuestionPool creates a new question pool
func NewQuestionPool() *QuestionPool {
	return &QuestionPool{
		Questions: make([]*TriviaQuestion, 0),
		Used:      make(map[string]bool),
		Index:     0,
	}
}

// AddQuestion adds a question to the pool
func (qp *QuestionPool) AddQuestion(q *TriviaQuestion) {
	qp.Questions = append(qp.Questions, q)
}

// GetNextQuestion gets the next question from the pool
func (qp *QuestionPool) GetNextQuestion() *TriviaQuestion {
	if len(qp.Questions) == 0 {
		return nil
	}

	// If we've used all questions, reset
	if qp.Index >= len(qp.Questions) {
		qp.Index = 0
		qp.Used = make(map[string]bool)
		// Shuffle questions for variety
		rand.Shuffle(len(qp.Questions), func(i, j int) {
			qp.Questions[i], qp.Questions[j] = qp.Questions[j], qp.Questions[i]
		})
	}

	// Find next unused question
	for qp.Index < len(qp.Questions) {
		q := qp.Questions[qp.Index]
		qp.Index++
		if !qp.Used[q.ID] {
			qp.Used[q.ID] = true
			return q
		}
	}

	// If somehow all are used, return first and reset
	qp.Index = 1
	qp.Used = make(map[string]bool)
	qp.Used[qp.Questions[0].ID] = true
	return qp.Questions[0]
}

// TriviaAPIResponse represents the API response wrapper
type TriviaAPIResponse struct {
	ResponseCode int                 `json:"response_code"`
	Results      []RawTriviaQuestion `json:"results"`
}

// LoadQuestionsFromJSON loads questions from JSON data
func LoadQuestionsFromJSON(data []byte) ([]*TriviaQuestion, error) {
	var response TriviaAPIResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	questions := make([]*TriviaQuestion, 0, len(response.Results))
	for _, rq := range response.Results {
		questions = append(questions, rq.ToTriviaQuestion())
	}

	return questions, nil
}
