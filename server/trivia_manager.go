package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
)

// TriviaManager handles loading and serving trivia questions with enhanced cycling
type TriviaManager struct {
	questions         map[string]map[string][]TriviaQuestion // category -> difficulty -> questions
	questionPools     map[string]map[string][]int            // category -> difficulty -> available question indices
	questionHistory   map[string]time.Time                   // questionID -> last asked time
	poolResetCounters map[string]map[string]int              // category -> difficulty -> reset counter
	mu                sync.RWMutex
}

// NewTriviaManager creates and initializes a new trivia manager
func NewTriviaManager() *TriviaManager {
	tm := &TriviaManager{
		questions:         make(map[string]map[string][]TriviaQuestion),
		questionPools:     make(map[string]map[string][]int),
		questionHistory:   make(map[string]time.Time),
		poolResetCounters: make(map[string]map[string]int),
	}

	if err := tm.loadAllQuestions(); err != nil {
		log.Printf("Error loading trivia questions: %v", err)
	}

	// Initialize question pools
	tm.initializeQuestionPools()

	// Start cleanup routine for question history
	go tm.cleanupQuestionHistory()

	return tm
}

// loadAllQuestions loads all trivia questions from the filesystem with enhanced error handling
func (tm *TriviaManager) loadAllQuestions() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	difficulties := []string{"easy", "medium", "hard"}
	totalLoaded := 0
	errors := make([]string, 0)

	for _, category := range constants.TriviaCategories {
		if tm.questions[category] == nil {
			tm.questions[category] = make(map[string][]TriviaQuestion)
		}

		for _, difficulty := range difficulties {
			filename := filepath.Join("trivia", category, fmt.Sprintf("%s.json", difficulty))
			questions, err := tm.loadQuestionsFromFile(filename, category, difficulty)
			if err != nil {
				errorMsg := fmt.Sprintf("Could not load %s: %v", filename, err)
				log.Printf("Warning: %s", errorMsg)
				errors = append(errors, errorMsg)
				continue
			}

			tm.questions[category][difficulty] = questions
			totalLoaded += len(questions)
			log.Printf("Loaded %d %s %s questions", len(questions), difficulty, category)
		}
	}

	log.Printf("Total questions loaded: %d", totalLoaded)

	if totalLoaded == 0 {
		return fmt.Errorf("no trivia questions loaded")
	}

	if len(errors) > 0 {
		log.Printf("Encountered %d errors during loading:", len(errors))
		for _, err := range errors {
			log.Printf("  - %s", err)
		}
	}

	return nil
}

// loadQuestionsFromFile loads questions from a single JSON file - ENHANCED validation
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
		return nil, fmt.Errorf("invalid JSON format: %v", err)
	}

	if response.ResponseCode != 0 {
		return nil, fmt.Errorf("API response error code: %d", response.ResponseCode)
	}

	if len(response.Results) == 0 {
		return nil, fmt.Errorf("no questions found in file")
	}

	questions := make([]TriviaQuestion, 0, len(response.Results))
	for i, q := range response.Results {
		// Validate question data
		if err := tm.validateQuestionData(q); err != nil {
			log.Printf("Skipping invalid question %d in %s: %v", i, filename, err)
			continue
		}

		// Clean HTML entities from question text
		questionText := cleanHTMLEntities(q.Question)
		correctAnswer := cleanHTMLEntities(q.CorrectAnswer)

		// Validate cleaned text
		if questionText == "" || correctAnswer == "" {
			log.Printf("Skipping question %d in %s: empty text after cleaning", i, filename)
			continue
		}

		// Create options array
		options := make([]string, 0, len(q.IncorrectAnswers)+1)
		options = append(options, correctAnswer)

		// Clean and validate incorrect answers
		validIncorrectAnswers := make([]string, 0, len(q.IncorrectAnswers))
		for _, incorrect := range q.IncorrectAnswers {
			cleaned := cleanHTMLEntities(incorrect)
			if cleaned != "" && cleaned != correctAnswer {
				options = append(options, cleaned)
				validIncorrectAnswers = append(validIncorrectAnswers, cleaned)
			}
		}

		// Ensure we have enough options
		if len(options) < 2 {
			log.Printf("Skipping question %d in %s: insufficient valid options", i, filename)
			continue
		}

		// Shuffle options
		rand.Shuffle(len(options), func(i, j int) {
			options[i], options[j] = options[j], options[i]
		})

		// Create unique ID that includes timestamp to prevent conflicts
		questions = append(questions, TriviaQuestion{
			ID:               fmt.Sprintf("%s_%s_%d_%d", category, difficulty, i, time.Now().UnixNano()%1000000),
			Text:             questionText,
			Category:         category,
			Difficulty:       difficulty,
			TimeLimit:        constants.TriviaAnswerTimeout, // FIXED: All questions get same base timeout
			Options:          options,
			CorrectAnswer:    correctAnswer,
			IncorrectAnswers: validIncorrectAnswers,
			IsSpecialty:      false, // Will be set when served as specialty
		})
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("no valid questions after processing")
	}

	log.Printf("Loaded %d valid questions from %s (all with %d second timeout)",
		len(questions), filename, constants.TriviaAnswerTimeout)

	return questions, nil
}

// validateQuestionData validates the basic structure of question data
func (tm *TriviaManager) validateQuestionData(q TriviaQuestionJSON) error {
	if q.Question == "" {
		return fmt.Errorf("empty question text")
	}

	if q.CorrectAnswer == "" {
		return fmt.Errorf("empty correct answer")
	}

	if len(q.IncorrectAnswers) == 0 {
		return fmt.Errorf("no incorrect answers provided")
	}

	if len(q.Question) > 500 {
		return fmt.Errorf("question text too long")
	}

	if len(q.CorrectAnswer) > 200 {
		return fmt.Errorf("correct answer too long")
	}

	// Check for duplicate answers
	allAnswers := append([]string{q.CorrectAnswer}, q.IncorrectAnswers...)
	seen := make(map[string]bool)
	for _, answer := range allAnswers {
		normalized := strings.ToLower(strings.TrimSpace(answer))
		if seen[normalized] {
			return fmt.Errorf("duplicate answers detected")
		}
		seen[normalized] = true
	}

	return nil
}

// initializeQuestionPools initializes the question pools for cycling
func (tm *TriviaManager) initializeQuestionPools() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for category, difficulties := range tm.questions {
		if tm.questionPools[category] == nil {
			tm.questionPools[category] = make(map[string][]int)
		}
		if tm.poolResetCounters[category] == nil {
			tm.poolResetCounters[category] = make(map[string]int)
		}

		for difficulty, questions := range difficulties {
			// Create index pool
			pool := make([]int, len(questions))
			for i := range pool {
				pool[i] = i
			}

			// Shuffle the initial pool
			rand.Shuffle(len(pool), func(i, j int) {
				pool[i], pool[j] = pool[j], pool[i]
			})

			tm.questionPools[category][difficulty] = pool
			tm.poolResetCounters[category][difficulty] = 0

			log.Printf("Initialized question pool for %s %s: %d questions", category, difficulty, len(pool))
		}
	}
}

// GetQuestion retrieves a question with FIXED time limits for specialty questions
func (tm *TriviaManager) GetQuestion(gameDifficulty string, playerSpecialties []string, askedQuestions map[string]bool) (*TriviaQuestion, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Determine question parameters
	var questionDifficulty string
	var isSpecialty bool
	var category string

	// Apply difficulty modifiers for specialty chance
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
		// Select from player's specialties
		availableSpecialties := tm.getAvailableSpecialtyCategories(playerSpecialties, gameDifficulty)
		if len(availableSpecialties) > 0 {
			category = availableSpecialties[rand.Intn(len(availableSpecialties))]
		} else {
			// Fallback to regular question if no specialty questions available
			isSpecialty = false
		}

		if isSpecialty {
			// Specialty questions are harder based on game difficulty
			switch gameDifficulty {
			case "easy":
				questionDifficulty = "medium"
			case "medium":
				questionDifficulty = "hard"
			case "hard":
				questionDifficulty = "hard"
			}
		}
	}

	if !isSpecialty {
		// Regular questions - apply difficulty modifiers
		diffMod := tm.getDifficultyModifiersForTrivia(gameDifficulty)

		if diffMod.TriviaModifier <= 0.8 {
			questionDifficulty = "easy"
		} else if diffMod.TriviaModifier >= 1.2 {
			questionDifficulty = "hard"
		} else {
			questionDifficulty = "medium"
		}

		// Select category from available categories with questions
		availableCategories := tm.getAvailableCategories(questionDifficulty)
		if len(availableCategories) == 0 {
			return nil, fmt.Errorf("no categories available for difficulty %s", questionDifficulty)
		}
		category = availableCategories[rand.Intn(len(availableCategories))]
	}

	// Get question from pool with cycling
	question, err := tm.getQuestionFromPool(category, questionDifficulty, askedQuestions)
	if err != nil {
		return nil, err
	}

	// Configure question settings
	question.IsSpecialty = isSpecialty
	if isSpecialty {
		// FIXED: Specialty questions get the SAME time limit as regular questions
		question.TimeLimit = constants.TriviaAnswerTimeout
		// Add specialty indicator to category display
		question.Category = question.Category + " (Specialty)"
	} else {
		// Regular questions also get the standard timeout
		question.TimeLimit = constants.TriviaAnswerTimeout
	}

	// REMOVED: No difficulty-based time modifiers applied to individual questions
	// The documentation specifies that all questions should have consistent time limits
	// Time modifications should only apply to phase durations, not individual question timeouts

	// Ensure minimum time limit safety check
	if question.TimeLimit < 10 {
		question.TimeLimit = 10
	}

	// Record when this question was asked
	tm.questionHistory[question.ID] = time.Now()

	log.Printf("Generated %s question (specialty: %v) with %d second timeout for category %s",
		questionDifficulty, isSpecialty, question.TimeLimit, category)

	return question, nil
}

// getAvailableSpecialtyCategories returns specialty categories that have questions
func (tm *TriviaManager) getAvailableSpecialtyCategories(specialties []string, gameDifficulty string) []string {
	var available []string

	// Determine what difficulty to look for
	var targetDifficulty string
	switch gameDifficulty {
	case "easy":
		targetDifficulty = "medium"
	case "medium":
		targetDifficulty = "hard"
	case "hard":
		targetDifficulty = "hard"
	}

	for _, specialty := range specialties {
		if tm.questions[specialty] != nil &&
			tm.questions[specialty][targetDifficulty] != nil &&
			len(tm.questions[specialty][targetDifficulty]) > 0 {
			available = append(available, specialty)
		}
	}

	return available
}

// getQuestionFromPool gets a question from the cycling pool
func (tm *TriviaManager) getQuestionFromPool(category, difficulty string, recentlyAsked map[string]bool) (*TriviaQuestion, error) {
	// Check if category and difficulty exist
	if tm.questions[category] == nil || tm.questions[category][difficulty] == nil {
		return nil, fmt.Errorf("no questions available for category %s difficulty %s", category, difficulty)
	}

	questions := tm.questions[category][difficulty]
	if len(questions) == 0 {
		return nil, fmt.Errorf("no questions in pool for category %s difficulty %s", category, difficulty)
	}

	pool := tm.questionPools[category][difficulty]

	// If pool is empty, automatically reset it (AUTOMATIC CYCLING)
	if len(pool) == 0 {
		tm.resetQuestionPool(category, difficulty)
		pool = tm.questionPools[category][difficulty]
		log.Printf("Automatically cycled question pool for %s %s (cycle #%d) - questions will repeat with new order",
			category, difficulty, tm.poolResetCounters[category][difficulty])
	}

	// Find a question that hasn't been recently asked
	var selectedIndex int
	var selectedQuestion *TriviaQuestion
	attempts := 0
	maxAttempts := min(len(pool), 10) // Limit attempts to prevent infinite loop

	for attempts < maxAttempts {
		// Take question from front of pool
		selectedIndex = pool[0]
		pool = pool[1:] // Remove from pool

		question := questions[selectedIndex]

		// Check if this question was recently asked
		if !recentlyAsked[question.ID] {
			selectedQuestion = &question
			break
		}

		// If recently asked, put it at the back of the pool and try next
		pool = append(pool, selectedIndex)
		attempts++
	}

	// Update the pool
	tm.questionPools[category][difficulty] = pool

	// If we couldn't find an unasked question, use the first available
	// This ensures we always return a question even if all were recently asked
	if selectedQuestion == nil {
		if len(pool) == 0 {
			// Pool was emptied during search, reset and try again
			tm.resetQuestionPool(category, difficulty)
			pool = tm.questionPools[category][difficulty]
			if len(pool) == 0 {
				return nil, fmt.Errorf("no questions available after reset for %s %s", category, difficulty)
			}
		}

		selectedIndex = pool[0]
		pool = pool[1:]
		tm.questionPools[category][difficulty] = pool
		question := questions[selectedIndex]
		selectedQuestion = &question
		log.Printf("All questions in %s %s recently asked, reusing question (automatic cycling active)", category, difficulty)
	}

	// Return a copy to prevent modification of the original
	questionCopy := *selectedQuestion
	return &questionCopy, nil
}

// resetQuestionPool resets a question pool for automatic cycling
func (tm *TriviaManager) resetQuestionPool(category, difficulty string) {
	questions := tm.questions[category][difficulty]
	if len(questions) == 0 {
		log.Printf("ERROR: Cannot reset pool for %s %s - no questions available", category, difficulty)
		return
	}

	pool := make([]int, len(questions))
	for i := range pool {
		pool[i] = i
	}

	// Shuffle the pool for variety in question order
	rand.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})

	tm.questionPools[category][difficulty] = pool
	tm.poolResetCounters[category][difficulty]++

	log.Printf("Reset question pool for %s %s with %d questions (cycle #%d)",
		category, difficulty, len(pool), tm.poolResetCounters[category][difficulty])
}

// getAvailableCategories returns categories that have questions for the given difficulty
func (tm *TriviaManager) getAvailableCategories(difficulty string) []string {
	var available []string
	for category, difficulties := range tm.questions {
		if difficulties[difficulty] != nil && len(difficulties[difficulty]) > 0 {
			available = append(available, category)
		}
	}
	return available
}

// cleanupQuestionHistory removes old question history entries
func (tm *TriviaManager) cleanupQuestionHistory() {
	ticker := time.NewTicker(10 * time.Minute) // Cleanup every 10 minutes
	defer ticker.Stop()

	for range ticker.C {
		tm.mu.Lock()
		cutoff := time.Now().Add(-30 * time.Minute) // Remove entries older than 30 minutes
		cleaned := 0
		for questionID, askTime := range tm.questionHistory {
			if askTime.Before(cutoff) {
				delete(tm.questionHistory, questionID)
				cleaned++
			}
		}
		if cleaned > 0 {
			log.Printf("Cleaned up %d old question history entries", cleaned)
		}
		tm.mu.Unlock()
	}
}

// Enhanced answer validation with consistent logging
func (tm *TriviaManager) ValidateAnswer(questionID, playerAnswer string) (bool, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Find the question
	for _, difficulties := range tm.questions {
		for _, questions := range difficulties {
			for _, question := range questions {
				if question.ID == questionID {
					correct := tm.compareAnswers(question.CorrectAnswer, playerAnswer)

					// Enhanced logging for debugging
					log.Printf("Answer validation: questionID=%s, correct=%s, player=%s, result=%v",
						questionID, question.CorrectAnswer, playerAnswer, correct)

					return correct, nil
				}
			}
		}
	}

	return false, fmt.Errorf("question not found: %s", questionID)
}

// Enhanced answer comparison with better variation handling
func (tm *TriviaManager) compareAnswers(correct, player string) bool {
	// Normalize both answers
	correctNorm := tm.normalizeAnswer(correct)
	playerNorm := tm.normalizeAnswer(player)

	// Exact match after normalization
	if correctNorm == playerNorm {
		return true
	}

	// Check for common variations
	if tm.checkAnswerVariations(correctNorm, playerNorm) {
		return true
	}

	// Additional fuzzy matching for numbers and common patterns
	return tm.checkFuzzyMatch(correctNorm, playerNorm)
}

// checkFuzzyMatch provides additional fuzzy matching for edge cases
func (tm *TriviaManager) checkFuzzyMatch(correct, player string) bool {
	// Handle numeric answers (e.g., "42" vs "forty-two")
	if tm.isNumericAnswer(correct) && tm.isNumericAnswer(player) {
		return tm.compareNumericAnswers(correct, player)
	}

	// Handle dates and years
	if tm.isDateAnswer(correct) && tm.isDateAnswer(player) {
		return tm.compareDateAnswers(correct, player)
	}

	// Handle percentage answers
	if strings.Contains(correct, "%") || strings.Contains(player, "%") {
		return tm.comparePercentageAnswers(correct, player)
	}

	return false
}

// Helper methods for fuzzy matching
func (tm *TriviaManager) isNumericAnswer(answer string) bool {
	// Check if answer contains primarily numbers
	numChars := 0
	totalChars := len(strings.ReplaceAll(answer, " ", ""))

	for _, char := range answer {
		if char >= '0' && char <= '9' {
			numChars++
		}
	}

	return totalChars > 0 && float64(numChars)/float64(totalChars) > 0.5
}

func (tm *TriviaManager) isDateAnswer(answer string) bool {
	// Check for common date patterns
	patterns := []string{"19", "20", "year", "century", "bc", "ad", "ce", "bce"}
	lowerAnswer := strings.ToLower(answer)

	for _, pattern := range patterns {
		if strings.Contains(lowerAnswer, pattern) {
			return true
		}
	}

	return false
}

func (tm *TriviaManager) compareNumericAnswers(correct, player string) bool {
	// Extract numbers from both answers and compare
	correctNums := tm.extractNumbers(correct)
	playerNums := tm.extractNumbers(player)

	// If both have same numbers, consider them equal
	if len(correctNums) > 0 && len(playerNums) > 0 {
		return correctNums[0] == playerNums[0]
	}

	return false
}

func (tm *TriviaManager) compareDateAnswers(correct, player string) bool {
	// Extract years from both answers
	correctYears := tm.extractYears(correct)
	playerYears := tm.extractYears(player)

	// If both contain the same year, consider them equal
	for _, cy := range correctYears {
		for _, py := range playerYears {
			if cy == py {
				return true
			}
		}
	}

	return false
}

func (tm *TriviaManager) comparePercentageAnswers(correct, player string) bool {
	// Extract percentage values
	correctPct := tm.extractPercentage(correct)
	playerPct := tm.extractPercentage(player)

	// Allow small tolerance for percentage answers
	if correctPct >= 0 && playerPct >= 0 {
		return abs(int(correctPct-playerPct)) <= 1 // 1% tolerance
	}

	return false
}

// Helper extraction methods
func (tm *TriviaManager) extractNumbers(text string) []int {
	numbers := []int{}
	current := ""

	for _, char := range text {
		if char >= '0' && char <= '9' {
			current += string(char)
		} else {
			if current != "" {
				if num, err := strconv.Atoi(current); err == nil {
					numbers = append(numbers, num)
				}
				current = ""
			}
		}
	}

	if current != "" {
		if num, err := strconv.Atoi(current); err == nil {
			numbers = append(numbers, num)
		}
	}

	return numbers
}

func (tm *TriviaManager) extractYears(text string) []int {
	numbers := tm.extractNumbers(text)
	years := []int{}

	for _, num := range numbers {
		// Consider 4-digit numbers between 1000-2100 as years
		if num >= 1000 && num <= 2100 {
			years = append(years, num)
		}
	}

	return years
}

func (tm *TriviaManager) extractPercentage(text string) float64 {
	// Look for numbers followed by % or "percent"
	text = strings.ToLower(text)

	if strings.Contains(text, "%") {
		// Extract number before %
		parts := strings.Split(text, "%")
		if len(parts) > 0 {
			numStr := strings.TrimSpace(parts[0])
			// Get the last word/number before %
			words := strings.Fields(numStr)
			if len(words) > 0 {
				if pct, err := strconv.ParseFloat(words[len(words)-1], 64); err == nil {
					return pct
				}
			}
		}
	}

	if strings.Contains(text, "percent") {
		// Extract number before "percent"
		parts := strings.Split(text, "percent")
		if len(parts) > 0 {
			numStr := strings.TrimSpace(parts[0])
			words := strings.Fields(numStr)
			if len(words) > 0 {
				if pct, err := strconv.ParseFloat(words[len(words)-1], 64); err == nil {
					return pct
				}
			}
		}
	}

	return -1 // No percentage found
}

// normalizeAnswer normalizes an answer for comparison
func (tm *TriviaManager) normalizeAnswer(answer string) string {
	// Convert to lowercase
	norm := strings.ToLower(answer)

	// Remove extra whitespace
	norm = strings.TrimSpace(norm)
	norm = strings.Join(strings.Fields(norm), " ")

	// Remove common punctuation
	norm = strings.ReplaceAll(norm, ".", "")
	norm = strings.ReplaceAll(norm, ",", "")
	norm = strings.ReplaceAll(norm, "!", "")
	norm = strings.ReplaceAll(norm, "?", "")
	norm = strings.ReplaceAll(norm, "'", "")
	norm = strings.ReplaceAll(norm, "\"", "")
	norm = strings.ReplaceAll(norm, "(", "")
	norm = strings.ReplaceAll(norm, ")", "")

	return norm
}

// checkAnswerVariations checks for common answer variations
func (tm *TriviaManager) checkAnswerVariations(correct, player string) bool {
	// Check if player answer is contained in correct answer or vice versa
	// This helps with answers like "The United States" vs "United States"
	if len(correct) > len(player) && strings.Contains(correct, player) && len(player) > 3 {
		return true
	}
	if len(player) > len(correct) && strings.Contains(player, correct) && len(correct) > 3 {
		return true
	}

	// Check for common abbreviations and variations
	variations := map[string][]string{
		"united states":  {"usa", "us", "america", "united states of america"},
		"united kingdom": {"uk", "britain", "great britain", "england"},
		"soviet union":   {"ussr", "russia"},
		"world war":      {"ww", "world war i", "world war ii", "wwi", "wwii"},
		"doctor":         {"dr", "doc"},
		"mount":          {"mt"},
		"saint":          {"st"},
		"president":      {"pres"},
		"association":    {"assoc"},
		"corporation":    {"corp"},
		"company":        {"co"},
		"incorporated":   {"inc"},
		"limited":        {"ltd"},
	}

	for canonical, vars := range variations {
		if strings.Contains(correct, canonical) {
			for _, variant := range vars {
				if strings.Contains(player, variant) {
					return true
				}
			}
		}
		// Check reverse too
		for _, variant := range vars {
			if strings.Contains(correct, variant) && strings.Contains(player, canonical) {
				return true
			}
		}
	}

	return false
}

// getDifficultyModifiersForTrivia gets difficulty modifiers for trivia
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

// GetQuestionByID retrieves a question by ID with enhanced error handling
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

// GetCategoryStats returns enhanced statistics about available questions
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

// GetPoolStats returns statistics about question pools
func (tm *TriviaManager) GetPoolStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	poolStats := make(map[string]interface{})

	for category, difficulties := range tm.questionPools {
		categoryStats := make(map[string]interface{})
		for difficulty, pool := range difficulties {
			total := 0
			if tm.questions[category] != nil && tm.questions[category][difficulty] != nil {
				total = len(tm.questions[category][difficulty])
			}

			categoryStats[difficulty] = map[string]interface{}{
				"remaining":   len(pool),
				"total":       total,
				"used":        total - len(pool),
				"cycles":      tm.poolResetCounters[category][difficulty],
				"utilization": float64(total-len(pool)) / float64(max(total, 1)),
			}
		}
		poolStats[category] = categoryStats
	}

	return poolStats
}

// ReloadQuestions reloads questions from filesystem with enhanced error handling
func (tm *TriviaManager) ReloadQuestions() error {
	log.Println("Reloading trivia questions...")

	// Create new structures
	newQuestions := make(map[string]map[string][]TriviaQuestion)
	newPools := make(map[string]map[string][]int)
	newCounters := make(map[string]map[string]int)

	difficulties := []string{"easy", "medium", "hard"}
	totalLoaded := 0

	for _, category := range constants.TriviaCategories {
		newQuestions[category] = make(map[string][]TriviaQuestion)
		newPools[category] = make(map[string][]int)
		newCounters[category] = make(map[string]int)

		for _, difficulty := range difficulties {
			filename := filepath.Join("trivia", category, fmt.Sprintf("%s.json", difficulty))
			questions, err := tm.loadQuestionsFromFile(filename, category, difficulty)
			if err != nil {
				log.Printf("Warning: Could not reload %s: %v", filename, err)
				continue
			}

			newQuestions[category][difficulty] = questions
			totalLoaded += len(questions)

			// Initialize new pool
			pool := make([]int, len(questions))
			for i := range pool {
				pool[i] = i
			}
			rand.Shuffle(len(pool), func(i, j int) {
				pool[i], pool[j] = pool[j], pool[i]
			})

			newPools[category][difficulty] = pool
			newCounters[category][difficulty] = 0

			log.Printf("Reloaded %d %s %s questions", len(questions), difficulty, category)
		}
	}

	if totalLoaded == 0 {
		return fmt.Errorf("no questions loaded during reload")
	}

	// Replace old data atomically
	tm.mu.Lock()
	tm.questions = newQuestions
	tm.questionPools = newPools
	tm.poolResetCounters = newCounters
	// Clear question history to avoid stale references
	tm.questionHistory = make(map[string]time.Time)
	tm.mu.Unlock()

	log.Printf("Trivia questions reloaded successfully: %d total questions", totalLoaded)
	return nil
}

// GetSummaryStats returns comprehensive statistics
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
		"historySize":         len(tm.questionHistory),
		"poolStats":           tm.GetPoolStats(),
		"cycling": map[string]interface{}{
			"enabled":     true,
			"automatic":   true,
			"cycleCount":  tm.getTotalCycles(),
			"description": "Questions automatically cycle when pools are exhausted - no manual refresh needed",
		},
	}
}

// getTotalCycles returns the total number of cycles across all categories
func (tm *TriviaManager) getTotalCycles() int {
	total := 0
	for _, difficulties := range tm.poolResetCounters {
		for _, count := range difficulties {
			total += count
		}
	}
	return total
}

// Utility methods

func (tm *TriviaManager) GetQuestionsCount(category, difficulty string) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.questions[category] == nil || tm.questions[category][difficulty] == nil {
		return 0
	}

	return len(tm.questions[category][difficulty])
}

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

func (tm *TriviaManager) IsCategorySupported(category string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, exists := tm.questions[category]
	return exists
}

func (tm *TriviaManager) GetAvailableCategories() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	categories := make([]string, 0, len(tm.questions))
	for category := range tm.questions {
		categories = append(categories, category)
	}
	return categories
}

// Utility functions

// cleanHTMLEntities removes HTML entities from text
func cleanHTMLEntities(text string) string {
	replacements := map[string]string{
		"&amp;":     "&",
		"&lt;":      "<",
		"&gt;":      ">",
		"&quot;":    "\"",
		"&#039;":    "'",
		"&rsquo;":   "'",
		"&lsquo;":   "'",
		"&rdquo;":   "\"",
		"&ldquo;":   "\"",
		"&ouml;":    "ö",
		"&auml;":    "ä",
		"&uuml;":    "ü",
		"&Ouml;":    "Ö",
		"&Auml;":    "Ä",
		"&Uuml;":    "Ü",
		"&szlig;":   "ß",
		"&eacute;":  "é",
		"&egrave;":  "è",
		"&ecirc;":   "ê",
		"&euml;":    "ë",
		"&aacute;":  "á",
		"&agrave;":  "à",
		"&acirc;":   "â",
		"&iacute;":  "í",
		"&igrave;":  "ì",
		"&icirc;":   "î",
		"&iuml;":    "ï",
		"&oacute;":  "ó",
		"&ograve;":  "ò",
		"&ocirc;":   "ô",
		"&otilde;":  "õ",
		"&uacute;":  "ú",
		"&ugrave;":  "ù",
		"&ucirc;":   "û",
		"&ccedil;":  "ç",
		"&ntilde;":  "ñ",
		"&nbsp;":    " ",
		"&hellip;":  "...",
		"&mdash;":   "—",
		"&ndash;":   "–",
		"&laquo;":   "«",
		"&raquo;":   "»",
		"&deg;":     "°",
		"&copy;":    "©",
		"&reg;":     "®",
		"&trade;":   "™",
		"&frac12;":  "½",
		"&frac14;":  "¼",
		"&frac34;":  "¾",
		"&plusmn;":  "±",
		"&times;":   "×",
		"&divide;":  "÷",
		"&micro;":   "μ",
		"&alpha;":   "α",
		"&beta;":    "β",
		"&gamma;":   "γ",
		"&delta;":   "δ",
		"&epsilon;": "ε",
		"&theta;":   "θ",
		"&lambda;":  "λ",
		"&mu;":      "μ",
		"&pi;":      "π",
		"&sigma;":   "σ",
		"&phi;":     "φ",
		"&omega;":   "ω",
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
