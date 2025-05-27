package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TriviaManager handles loading and serving trivia questions
type TriviaManager struct {
	questions map[string]map[string][]TriviaQuestion // category -> difficulty -> questions
	mu        sync.RWMutex
}

// NewTriviaManager creates and initializes a new trivia manager
func NewTriviaManager() *TriviaManager {
	tm := &TriviaManager{
		questions: make(map[string]map[string][]TriviaQuestion),
	}

	if err := tm.loadAllQuestions(); err != nil {
		log.Printf("Error loading trivia questions: %v", err)
	}

	return tm
}

// loadAllQuestions loads all trivia questions from the filesystem
func (tm *TriviaManager) loadAllQuestions() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	difficulties := []string{"easy", "medium", "hard"}

	for _, category := range TriviaCategories {
		if tm.questions[category] == nil {
			tm.questions[category] = make(map[string][]TriviaQuestion)
		}

		for _, difficulty := range difficulties {
			filename := filepath.Join("trivia", category, fmt.Sprintf("%s.json", difficulty))
			questions, err := tm.loadQuestionsFromFile(filename)
			if err != nil {
				log.Printf("Warning: Could not load %s: %v", filename, err)
				continue
			}

			tm.questions[category][difficulty] = questions
			log.Printf("Loaded %d %s %s questions", len(questions), difficulty, category)
		}
	}

	return nil
}

// loadQuestionsFromFile loads questions from a single JSON file
func (tm *TriviaManager) loadQuestionsFromFile(filename string) ([]TriviaQuestion, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var response struct {
		ResponseCode int                  `json:"response_code"`
		Results      []TriviaQuestionJSON `json:"results"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	questions := make([]TriviaQuestion, 0, len(response.Results))
	for i, q := range response.Results {
		// Clean HTML entities from question text
		questionText := cleanHTMLEntities(q.Question)
		correctAnswer := cleanHTMLEntities(q.CorrectAnswer)

		// Create options array
		options := make([]string, 0, len(q.IncorrectAnswers)+1)
		options = append(options, correctAnswer)
		for _, incorrect := range q.IncorrectAnswers {
			options = append(options, cleanHTMLEntities(incorrect))
		}

		// Shuffle options
		rand.Shuffle(len(options), func(i, j int) {
			options[i], options[j] = options[j], options[i]
		})

		questions = append(questions, TriviaQuestion{
			ID:               fmt.Sprintf("%s_%s_%d", q.Category, q.Difficulty, i),
			Text:             questionText,
			Category:         normalizeCategory(q.Category),
			Difficulty:       q.Difficulty,
			TimeLimit:        TriviaAnswerTimeout,
			Options:          options,
			CorrectAnswer:    correctAnswer,
			IncorrectAnswers: q.IncorrectAnswers,
		})
	}

	return questions, nil
}

// GetQuestion retrieves a trivia question based on game difficulty and player specialties
func (tm *TriviaManager) GetQuestion(gameDifficulty string, playerSpecialties []string, askedQuestions map[string]bool) (*TriviaQuestion, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Determine question difficulty based on game difficulty
	var questionDifficulty string
	var isSpecialty bool

	// Check if we should ask a specialty question (30% chance if player has specialties)
	if len(playerSpecialties) > 0 && rand.Float32() < 0.3 {
		isSpecialty = true
		// Specialty questions are harder
		switch gameDifficulty {
		case "easy", "medium":
			questionDifficulty = "medium"
		case "hard":
			questionDifficulty = "hard"
		}
	} else {
		// Regular questions
		switch gameDifficulty {
		case "easy", "medium":
			questionDifficulty = "easy"
		case "hard":
			questionDifficulty = "medium"
		}
	}

	// Select category
	var category string
	if isSpecialty {
		category = playerSpecialties[rand.Intn(len(playerSpecialties))]
	} else {
		// Random category from all available
		categories := make([]string, 0, len(tm.questions))
		for cat := range tm.questions {
			categories = append(categories, cat)
		}
		if len(categories) == 0 {
			return nil, fmt.Errorf("no categories available")
		}
		category = categories[rand.Intn(len(categories))]
	}

	// Get questions for this category and difficulty
	if tm.questions[category] == nil || tm.questions[category][questionDifficulty] == nil {
		return nil, fmt.Errorf("no questions available for category %s difficulty %s", category, questionDifficulty)
	}

	availableQuestions := tm.questions[category][questionDifficulty]
	if len(availableQuestions) == 0 {
		return nil, fmt.Errorf("no questions in pool")
	}

	// Find an unasked question
	unaskedQuestions := make([]TriviaQuestion, 0)
	for _, q := range availableQuestions {
		if !askedQuestions[q.ID] {
			unaskedQuestions = append(unaskedQuestions, q)
		}
	}

	// If all questions have been asked, reset and use any question
	if len(unaskedQuestions) == 0 {
		unaskedQuestions = availableQuestions
		// Clear asked questions for this category
		for id := range askedQuestions {
			for _, q := range availableQuestions {
				if q.ID == id {
					delete(askedQuestions, id)
					break
				}
			}
		}
	}

	// Select random question
	selected := unaskedQuestions[rand.Intn(len(unaskedQuestions))]

	// Mark as specialty if applicable
	if isSpecialty {
		selected.Category = selected.Category + " (Specialty)"
	}

	return &selected, nil
}

// cleanHTMLEntities removes common HTML entities from text
func cleanHTMLEntities(text string) string {
	replacements := map[string]string{
		"&amp;":   "&",
		"&lt;":    "<",
		"&gt;":    ">",
		"&quot;":  "\"",
		"&#039;":  "'",
		"&rsquo;": "'",
		"&lsquo;": "'",
		"&rdquo;": "\"",
		"&ldquo;": "\"",
		"&ouml;":  "ö",
		"&auml;":  "ä",
		"&uuml;":  "ü",
		"&Ouml;":  "Ö",
		"&Auml;":  "Ä",
		"&Uuml;":  "Ü",
	}

	result := text
	for entity, replacement := range replacements {
		result = strings.ReplaceAll(result, entity, replacement)
	}

	return result
}

// normalizeCategory converts category names to match our expected format
func normalizeCategory(category string) string {
	// Convert to lowercase and replace spaces with underscores
	normalized := strings.ToLower(category)
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, "-", "_")

	// Map common variations
	categoryMap := map[string]string{
		"general_knowledge":         "general",
		"entertainment_video_games": "video_games",
		"entertainment_music":       "music",
		"science_nature":            "science",
		"science_&_nature":          "science",
	}

	if mapped, ok := categoryMap[normalized]; ok {
		return mapped
	}

	return normalized
}

// GetCategoryStats returns statistics about available questions
func (tm *TriviaManager) GetCategoryStats() map[string]map[string]int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := make(map[string]map[string]int)

	for category, difficulties := range tm.questions {
		stats[category] = make(map[string]int)
		for difficulty, questions := range difficulties {
			stats[category][difficulty] = len(questions)
		}
	}

	return stats
}
