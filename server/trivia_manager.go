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

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
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

	for _, category := range constants.TriviaCategories {
		if tm.questions[category] == nil {
			tm.questions[category] = make(map[string][]TriviaQuestion)
		}

		for _, difficulty := range difficulties {
			filename := filepath.Join("trivia", category, fmt.Sprintf("%s.json", difficulty))
			questions, err := tm.loadQuestionsFromFile(filename, category, difficulty)
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
func (tm *TriviaManager) loadQuestionsFromFile(filename, category, difficulty string) ([]TriviaQuestion, error) {
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
			ID:               fmt.Sprintf("%s_%s_%d", category, difficulty, i),
			Text:             questionText,
			Category:         category,
			Difficulty:       difficulty,
			TimeLimit:        constants.TriviaAnswerTimeout,
			Options:          options,
			CorrectAnswer:    correctAnswer,
			IncorrectAnswers: q.IncorrectAnswers,
			IsSpecialty:      false, // Will be set when served as specialty
		})
	}

	return questions, nil
}

// ENHANCED: GetQuestion with proper difficulty modifiers and specialty marking
func (tm *TriviaManager) GetQuestion(gameDifficulty string, playerSpecialties []string, askedQuestions map[string]bool) (*TriviaQuestion, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Determine question difficulty and specialty based on game difficulty
	var questionDifficulty string
	var isSpecialty bool
	var category string

	// IMPLEMENTED: Apply difficulty modifiers for specialty chance
	specialtyChance := 0.3 // Base 30% chance
	switch gameDifficulty {
	case "easy":
		specialtyChance = 0.2 // 20% for easy
	case "hard":
		specialtyChance = 0.4 // 40% for hard
	}

	// Check if we should ask a specialty question
	if len(playerSpecialties) > 0 && rand.Float32() < float32(specialtyChance) {
		isSpecialty = true
		category = playerSpecialties[rand.Intn(len(playerSpecialties))]

		// IMPLEMENTED: Specialty questions are harder based on game difficulty
		switch gameDifficulty {
		case "easy":
			questionDifficulty = "medium"
		case "medium":
			questionDifficulty = "hard"
		case "hard":
			questionDifficulty = "hard" // Already at max
		}
	} else {
		// Regular questions - apply difficulty modifiers
		diffMod := tm.getDifficultyModifiersForTrivia(gameDifficulty)

		// Choose difficulty based on game settings and modifiers
		if diffMod.TriviaModifier <= 0.8 {
			questionDifficulty = "easy"
		} else if diffMod.TriviaModifier >= 1.2 {
			questionDifficulty = "hard"
		} else {
			questionDifficulty = "medium"
		}

		// Random category from all available
		categories := make([]string, 0, len(tm.questions))
		for cat := range tm.questions {
			if len(tm.questions[cat][questionDifficulty]) > 0 {
				categories = append(categories, cat)
			}
		}
		if len(categories) == 0 {
			// Fallback to medium if requested difficulty not available
			questionDifficulty = "medium"
			for cat := range tm.questions {
				if len(tm.questions[cat][questionDifficulty]) > 0 {
					categories = append(categories, cat)
				}
			}
		}
		if len(categories) == 0 {
			return nil, fmt.Errorf("no categories available for any difficulty")
		}
		category = categories[rand.Intn(len(categories))]
	}

	// Get questions for this category and difficulty
	if tm.questions[category] == nil || tm.questions[category][questionDifficulty] == nil {
		return nil, fmt.Errorf("no questions available for category %s difficulty %s", category, questionDifficulty)
	}

	availableQuestions := tm.questions[category][questionDifficulty]
	if len(availableQuestions) == 0 {
		return nil, fmt.Errorf("no questions in pool for category %s difficulty %s", category, questionDifficulty)
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
		// Clear asked questions for this category and difficulty
		for id := range askedQuestions {
			for _, q := range availableQuestions {
				if q.ID == id {
					delete(askedQuestions, id)
					break
				}
			}
		}
		log.Printf("Reset question pool for category %s difficulty %s", category, questionDifficulty)
	}

	// Select random question
	selected := unaskedQuestions[rand.Intn(len(unaskedQuestions))]

	// IMPLEMENTED: Mark as specialty and adjust time limit
	selected.IsSpecialty = isSpecialty
	if isSpecialty {
		// Specialty questions get more time
		selected.TimeLimit = int(float64(constants.TriviaAnswerTimeout) * 1.5)
		// Add specialty indicator to category display
		selected.Category = selected.Category + " (Specialty)"
	}

	// Apply difficulty-based time modifiers
	diffMod := tm.getDifficultyModifiersForTrivia(gameDifficulty)
	selected.TimeLimit = int(float64(selected.TimeLimit) * diffMod.TimeLimitModifier)

	// Ensure minimum time limit
	if selected.TimeLimit < 10 {
		selected.TimeLimit = 10
	}

	return &selected, nil
}

// IMPLEMENTED: Get difficulty modifiers for trivia
func (tm *TriviaManager) getDifficultyModifiersForTrivia(gameDifficulty string) constants.DifficultyModifiers {
	switch gameDifficulty {
	case "easy":
		return constants.EasyMode
	case "hard":
		return constants.HardMode
	default:
		return constants.MediumMode
	}
}

// IMPLEMENTED: Validate answer against stored correct answer
func (tm *TriviaManager) ValidateAnswer(questionID, playerAnswer string) (bool, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Find the question
	for _, difficulties := range tm.questions {
		for _, questions := range difficulties {
			for _, question := range questions {
				if question.ID == questionID {
					// Case-insensitive comparison, trimmed
					correctAnswer := strings.TrimSpace(strings.ToLower(question.CorrectAnswer))
					playerAnswerClean := strings.TrimSpace(strings.ToLower(playerAnswer))
					return correctAnswer == playerAnswerClean, nil
				}
			}
		}
	}

	return false, fmt.Errorf("question not found: %s", questionID)
}

// IMPLEMENTED: Get question by ID for validation
func (tm *TriviaManager) GetQuestionByID(questionID string) (*TriviaQuestion, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	for _, difficulties := range tm.questions {
		for _, questions := range difficulties {
			for _, question := range questions {
				if question.ID == questionID {
					// Return a copy to prevent modification
					questionCopy := question
					return &questionCopy, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("question not found: %s", questionID)
}

// cleanHTMLEntities removes common HTML entities from text
func cleanHTMLEntities(text string) string {
	replacements := map[string]string{
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&#039;":   "'",
		"&rsquo;":  "'",
		"&lsquo;":  "'",
		"&rdquo;":  "\"",
		"&ldquo;":  "\"",
		"&ouml;":   "ö",
		"&auml;":   "ä",
		"&uuml;":   "ü",
		"&Ouml;":   "Ö",
		"&Auml;":   "Ä",
		"&Uuml;":   "Ü",
		"&szlig;":  "ß",
		"&eacute;": "é",
		"&egrave;": "è",
		"&ecirc;":  "ê",
		"&euml;":   "ë",
		"&aacute;": "á",
		"&agrave;": "à",
		"&acirc;":  "â",
		"&iacute;": "í",
		"&igrave;": "ì",
		"&icirc;":  "î",
		"&iuml;":   "ï",
		"&oacute;": "ó",
		"&ograve;": "ò",
		"&ocirc;":  "ô",
		"&otilde;": "õ",
		"&uacute;": "ú",
		"&ugrave;": "ù",
		"&ucirc;":  "û",
		"&ccedil;": "ç",
		"&ntilde;": "ñ",
		"&nbsp;":   " ",
		"&hellip;": "...",
		"&mdash;":  "—",
		"&ndash;":  "–",
		"&laquo;":  "«",
		"&raquo;":  "»",
		"&deg;":    "°",
		"&copy;":   "©",
		"&reg;":    "®",
		"&trade;":  "™",
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

	// Map common variations to our supported categories
	categoryMap := map[string]string{
		"general_knowledge":                  "general",
		"entertainment_video_games":          "video_games",
		"entertainment_music":                "music",
		"science_nature":                     "science",
		"science_&_nature":                   "science",
		"entertainment_books":                "general",
		"entertainment_film":                 "general",
		"entertainment_television":           "general",
		"sports":                             "general",
		"art":                                "general",
		"celebrities":                        "general",
		"animals":                            "science",
		"vehicles":                           "general",
		"entertainment_comics":               "general",
		"entertainment_japanese_anime_manga": "general",
		"entertainment_cartoons_animations":  "general",
		"mythology":                          "history",
		"politics":                           "history",
	}

	if mapped, ok := categoryMap[normalized]; ok {
		return mapped
	}

	// Check if it's one of our supported categories
	for _, supportedCat := range constants.TriviaCategories {
		if normalized == supportedCat {
			return normalized
		}
	}

	// Default to general if we don't recognize the category
	return "general"
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

// IMPLEMENTED: Get questions by difficulty and category for validation
func (tm *TriviaManager) GetQuestionsCount(category, difficulty string) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.questions[category] == nil || tm.questions[category][difficulty] == nil {
		return 0
	}

	return len(tm.questions[category][difficulty])
}

// IMPLEMENTED: Validate if a question ID exists
func (tm *TriviaManager) ValidateQuestion(questionID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	for _, difficulties := range tm.questions {
		for _, questions := range difficulties {
			for _, question := range questions {
				if question.ID == questionID {
					return true
				}
			}
		}
	}
	return false
}

// IMPLEMENTED: Get total questions count across all categories and difficulties
func (tm *TriviaManager) GetTotalQuestionsCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	total := 0
	for _, difficulties := range tm.questions {
		for _, questions := range difficulties {
			total += len(questions)
		}
	}
	return total
}

// IMPLEMENTED: Get questions for a specific category and difficulty
func (tm *TriviaManager) GetQuestionsByCategory(category, difficulty string) []TriviaQuestion {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.questions[category] == nil || tm.questions[category][difficulty] == nil {
		return []TriviaQuestion{}
	}

	// Return a copy to prevent external modification
	questions := make([]TriviaQuestion, len(tm.questions[category][difficulty]))
	copy(questions, tm.questions[category][difficulty])
	return questions
}

// IMPLEMENTED: Check if category is supported
func (tm *TriviaManager) IsCategorySupported(category string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, exists := tm.questions[category]
	return exists
}

// IMPLEMENTED: Get all available categories
func (tm *TriviaManager) GetAvailableCategories() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	categories := make([]string, 0, len(tm.questions))
	for category := range tm.questions {
		categories = append(categories, category)
	}
	return categories
}

// IMPLEMENTED: Reload questions from filesystem (useful for updates without restart)
func (tm *TriviaManager) ReloadQuestions() error {
	log.Println("Reloading trivia questions...")

	// Create new questions map
	newQuestions := make(map[string]map[string][]TriviaQuestion)
	difficulties := []string{"easy", "medium", "hard"}

	for _, category := range constants.TriviaCategories {
		newQuestions[category] = make(map[string][]TriviaQuestion)

		for _, difficulty := range difficulties {
			filename := filepath.Join("trivia", category, fmt.Sprintf("%s.json", difficulty))
			questions, err := tm.loadQuestionsFromFile(filename, category, difficulty)
			if err != nil {
				log.Printf("Warning: Could not reload %s: %v", filename, err)
				continue
			}

			newQuestions[category][difficulty] = questions
			log.Printf("Reloaded %d %s %s questions", len(questions), difficulty, category)
		}
	}

	// Replace old questions with new ones atomically
	tm.mu.Lock()
	tm.questions = newQuestions
	tm.mu.Unlock()

	log.Println("Trivia questions reloaded successfully")
	return nil
}

// IMPLEMENTED: Get summary statistics
func (tm *TriviaManager) GetSummaryStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	totalQuestions := 0
	categoryCounts := make(map[string]int)
	difficultyCounts := make(map[string]int)

	for category, difficulties := range tm.questions {
		categoryTotal := 0
		for difficulty, questions := range difficulties {
			count := len(questions)
			totalQuestions += count
			categoryTotal += count
			difficultyCounts[difficulty] += count
		}
		categoryCounts[category] = categoryTotal
	}

	return map[string]interface{}{
		"totalQuestions":      totalQuestions,
		"categoryCounts":      categoryCounts,
		"difficultyCounts":    difficultyCounts,
		"supportedCategories": constants.TriviaCategories,
	}
}
