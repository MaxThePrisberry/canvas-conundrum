package services

import (
	"canvas-conundrum/constants"
	"canvas-conundrum/models"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"sync"
)

// TriviaService manages trivia questions and delivery
type TriviaService struct {
	mu       sync.RWMutex
	pools    map[string]*models.QuestionPool // Key: "category_difficulty"
	basePath string                          // Base path for trivia files
}

// NewTriviaService creates a new trivia service
func NewTriviaService() *TriviaService {
	return &TriviaService{
		pools:    make(map[string]*models.QuestionPool),
		basePath: "./trivia",
	}
}

// LoadQuestions loads all trivia questions from JSON files
func (ts *TriviaService) LoadQuestions() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

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

	totalQuestions := 0

	for _, category := range categories {
		for _, difficulty := range difficulties {
			poolKey := ts.getPoolKey(category, difficulty)
			pool := models.NewQuestionPool()

			// Load questions from file
			filename := fmt.Sprintf("%s/%s/%s.json", ts.basePath, category, difficulty)
			questions, err := ts.loadQuestionsFromFile(filename)
			if err != nil {
				log.Printf("Warning: Failed to load questions from %s: %v", filename, err)
				continue
			}

			// Add questions to pool
			for _, q := range questions {
				q.Category = category
				q.Difficulty = difficulty
				pool.AddQuestion(q)
			}

			ts.pools[poolKey] = pool
			totalQuestions += len(questions)
			log.Printf("Loaded %d questions for %s/%s", len(questions), category, difficulty)
		}
	}

	log.Printf("Total trivia questions loaded: %d", totalQuestions)
	return nil
}

// loadQuestionsFromFile loads questions from a JSON file
func (ts *TriviaService) loadQuestionsFromFile(filename string) ([]*models.TriviaQuestion, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return models.LoadQuestionsFromJSON(data)
}

// getPoolKey generates a key for the question pool map
func (ts *TriviaService) getPoolKey(category models.TriviaCategory, difficulty models.TriviaDifficulty) string {
	return fmt.Sprintf("%s_%s", category, difficulty)
}

// GetQuestionsForRound gets trivia questions for all players for the current round
func (ts *TriviaService) GetQuestionsForRound(players map[string]*models.Player) map[string]*models.TriviaQuestion {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	questions := make(map[string]*models.TriviaQuestion)
	gameManager := GetGameInstance()
	game := gameManager.GetGame()

	for playerID, player := range players {
		if !player.IsActive {
			continue
		}

		// Determine if this should be a specialty question
		isSpecialty := ts.shouldBeSpecialty(player, game.Difficulty)

		var question *models.TriviaQuestion
		if isSpecialty && len(player.Specialties) > 0 {
			// Get specialty question
			question = ts.getSpecialtyQuestion(player, game.Difficulty)
		} else {
			// Get random question
			question = ts.getRandomQuestion(game.Difficulty)
		}

		if question != nil {
			// Mark if this is a specialty question for the player
			if isSpecialty && len(player.Specialties) > 0 {
				for _, specialty := range player.Specialties {
					if question.Category == specialty {
						question.IsSpecialty = true
						question.SpecialtyBonus = true
						break
					}
				}
			}
			questions[playerID] = question
		}
	}

	return questions
}

// shouldBeSpecialty determines if a player should get a specialty question
func (ts *TriviaService) shouldBeSpecialty(player *models.Player, difficulty models.DifficultyMode) bool {
	if len(player.Specialties) == 0 {
		return false
	}

	var probability float64
	switch difficulty {
	case models.DifficultyEasy:
		probability = constants.EasySpecialtyProbability
	case models.DifficultyHard:
		probability = constants.HardSpecialtyProbability
	default:
		probability = constants.MediumSpecialtyProbability
	}

	return rand.Float64() < probability
}

// getSpecialtyQuestion gets a specialty question for a player
func (ts *TriviaService) getSpecialtyQuestion(player *models.Player, gameDifficulty models.DifficultyMode) *models.TriviaQuestion {
	if len(player.Specialties) == 0 {
		return ts.getRandomQuestion(gameDifficulty)
	}

	// Pick a random specialty
	specialty := player.Specialties[rand.Intn(len(player.Specialties))]
	category := models.TriviaCategory(specialty)

	// Specialty questions are one difficulty level harder
	difficulty := ts.getHarderDifficulty(ts.mapGameDifficulty(gameDifficulty))

	poolKey := ts.getPoolKey(category, difficulty)
	pool, exists := ts.pools[poolKey]
	if !exists || pool == nil {
		// Fallback to random question
		return ts.getRandomQuestion(gameDifficulty)
	}

	return pool.GetNextQuestion()
}

// getRandomQuestion gets a random question based on difficulty
func (ts *TriviaService) getRandomQuestion(gameDifficulty models.DifficultyMode) *models.TriviaQuestion {
	difficulty := ts.mapGameDifficulty(gameDifficulty)

	// Get all categories
	categories := []models.TriviaCategory{
		models.CategoryGeneral,
		models.CategoryGeography,
		models.CategoryHistory,
		models.CategoryMusic,
		models.CategoryScience,
		models.CategoryVideoGames,
	}

	// Shuffle categories for variety
	rand.Shuffle(len(categories), func(i, j int) {
		categories[i], categories[j] = categories[j], categories[i]
	})

	// Try each category until we find a question
	for _, category := range categories {
		poolKey := ts.getPoolKey(category, difficulty)
		pool, exists := ts.pools[poolKey]
		if exists && pool != nil {
			if question := pool.GetNextQuestion(); question != nil {
				return question
			}
		}
	}

	// If no questions found at desired difficulty, try other difficulties
	difficulties := []models.TriviaDifficulty{
		models.DifficultyMediumTrivia,
		models.DifficultyEasyTrivia,
		models.DifficultyHardTrivia,
	}

	for _, diff := range difficulties {
		if diff == difficulty {
			continue // Already tried
		}
		for _, category := range categories {
			poolKey := ts.getPoolKey(category, diff)
			pool, exists := ts.pools[poolKey]
			if exists && pool != nil {
				if question := pool.GetNextQuestion(); question != nil {
					return question
				}
			}
		}
	}

	log.Println("Warning: No trivia questions available")
	return nil
}

// mapGameDifficulty maps game difficulty to trivia difficulty
func (ts *TriviaService) mapGameDifficulty(gameDifficulty models.DifficultyMode) models.TriviaDifficulty {
	switch gameDifficulty {
	case models.DifficultyEasy:
		return models.DifficultyEasyTrivia
	case models.DifficultyHard:
		return models.DifficultyHardTrivia
	default:
		return models.DifficultyMediumTrivia
	}
}

// getHarderDifficulty returns one difficulty level harder
func (ts *TriviaService) getHarderDifficulty(current models.TriviaDifficulty) models.TriviaDifficulty {
	switch current {
	case models.DifficultyEasyTrivia:
		return models.DifficultyMediumTrivia
	case models.DifficultyMediumTrivia:
		return models.DifficultyHardTrivia
	default:
		return models.DifficultyHardTrivia
	}
}

// ProcessAnswer processes a player's answer to a trivia question
func (ts *TriviaService) ProcessAnswer(playerID string, questionID string, selectedAnswer string, answerIndex int, timeElapsed float64) (*models.TriviaAnswer, int, models.TokenType) {
	gameManager := GetGameInstance()
	player, exists := gameManager.GetPlayer(playerID)
	if !exists {
		return nil, 0, ""
	}

	game := gameManager.GetGame()

	// Get the original question to validate answer
	question := ts.GetQuestionByID(questionID)
	if question == nil {
		return nil, 0, ""
	}

	// Check if answer is correct
	correct := question.CorrectAnswer == selectedAnswer

	// Create answer record
	answer := &models.TriviaAnswer{
		PlayerID:       playerID,
		QuestionID:     questionID,
		SelectedAnswer: selectedAnswer,
		AnswerIndex:    answerIndex,
		TimeElapsed:    timeElapsed,
		Correct:        correct,
	}

	// Calculate tokens earned
	tokensEarned := 0
	var tokenType models.TokenType

	if correct {
		// Base tokens
		tokensEarned = constants.BaseTokensPerCorrectAnswer

		// Get token type for current station
		if player.CurrentStation != "" {
			tokenType = ts.getTokenTypeForStation(player.CurrentStation)

			// Apply role bonus if at matching station
			if tokenType == player.Role.GetBonusTokenType() {
				tokensEarned = int(float64(tokensEarned) * constants.RoleResourceMultiplier)
			}
		}

		// Apply specialty bonus
		if question.IsSpecialty {
			tokensEarned = int(float64(tokensEarned) * constants.SpecialtyPointMultiplier)
		}

		// Apply difficulty modifier
		switch game.Difficulty {
		case models.DifficultyEasy:
			tokensEarned = int(float64(tokensEarned) * constants.EasyTimeMultiplier)
		case models.DifficultyHard:
			tokensEarned = int(float64(tokensEarned) * constants.HardTimeMultiplier)
		}
	}

	answer.TokensEarned = tokensEarned
	return answer, tokensEarned, tokenType
}

// getTokenTypeForStation returns the token type for a station
func (ts *TriviaService) getTokenTypeForStation(station interface{}) models.TokenType {
	// Handle both string and config.Station types
	var stationStr string
	switch v := station.(type) {
	case string:
		stationStr = v
	default:
		stationStr = fmt.Sprintf("%v", v)
	}

	switch stationStr {
	case "anchor":
		return models.TokenAnchor
	case "chronos":
		return models.TokenChronos
	case "guide":
		return models.TokenGuide
	case "clarity":
		return models.TokenClarity
	default:
		return ""
	}
}

// GetQuestionByID retrieves a question by its ID (for answer validation)
func (ts *TriviaService) GetQuestionByID(questionID string) *models.TriviaQuestion {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Search through all pools for the question
	for _, pool := range ts.pools {
		for _, question := range pool.Questions {
			if question.ID == questionID {
				return question
			}
		}
	}

	return nil
}
