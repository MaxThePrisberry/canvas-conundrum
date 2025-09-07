package services

import (
	"canvas-conundrum/config"
	"canvas-conundrum/models"
	"log"
	"sync"
	"time"
)

// AnalyticsService manages game analytics and scoring
type AnalyticsService struct {
	mu        sync.RWMutex
	analytics *models.GameAnalytics
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{}
}

// StartGame initializes analytics for a new game
func (as *AnalyticsService) StartGame(gameID string) {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.analytics = models.NewGameAnalytics(gameID)
	log.Printf("Analytics initialized for game %s", gameID)
}

// InitializePlayer initializes analytics for a player
func (as *AnalyticsService) InitializePlayer(playerID, playerName string) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.analytics == nil {
		return
	}

	as.analytics.PlayerAnalytics[playerID] = &models.PlayerAnalytics{
		PlayerID:           playerID,
		PlayerName:         playerName,
		TokensEarned:       make(map[models.TokenType]int),
		AccuracyByCategory: make(map[models.TriviaCategory]float64),
		StationPreferences: make(map[string]int),
		Achievements:       []string{},
	}
}

// RecordTriviaAnswer records a trivia answer
func (as *AnalyticsService) RecordTriviaAnswer(playerID string, category models.TriviaCategory,
	correct bool, responseTime float64, tokensEarned int, isSpecialty bool) {

	as.mu.Lock()
	defer as.mu.Unlock()

	if as.analytics == nil {
		return
	}

	playerAnalytics, exists := as.analytics.PlayerAnalytics[playerID]
	if !exists {
		return
	}

	// Update trivia stats
	playerAnalytics.TotalQuestions++
	if correct {
		playerAnalytics.CorrectAnswers++
	}

	// Update specialty stats
	if isSpecialty {
		playerAnalytics.SpecialtyQuestions++
		if correct {
			playerAnalytics.SpecialtyCorrect++
			playerAnalytics.SpecialtyBonus += tokensEarned
		}
	}

	// Update response time
	if playerAnalytics.AverageResponseTime == 0 {
		playerAnalytics.AverageResponseTime = responseTime
	} else {
		// Running average
		playerAnalytics.AverageResponseTime =
			(playerAnalytics.AverageResponseTime*float64(playerAnalytics.TotalQuestions-1) + responseTime) /
				float64(playerAnalytics.TotalQuestions)
	}

	// Update category stats
	if as.analytics.CategoryPerformance[category] == nil {
		as.analytics.CategoryPerformance[category] = &models.CategoryStats{}
	}
	as.analytics.CategoryPerformance[category].QuestionsAsked++
	if correct {
		as.analytics.CategoryPerformance[category].CorrectAnswers++
	}

	// Update resource gathering metrics
	as.analytics.ResourceGatheringMetrics.QuestionsAnswered++

	playerAnalytics.TotalTokens += tokensEarned
}

// RecordTokenCollection records token collection
func (as *AnalyticsService) RecordTokenCollection(playerID string, tokenType models.TokenType, amount int) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.analytics == nil {
		return
	}

	playerAnalytics, exists := as.analytics.PlayerAnalytics[playerID]
	if !exists {
		return
	}

	// Update player tokens
	if playerAnalytics.TokensEarned[tokenType] == 0 {
		playerAnalytics.TokensEarned[tokenType] = amount
	} else {
		playerAnalytics.TokensEarned[tokenType] += amount
	}

	// Update team tokens in resource metrics
	as.analytics.ResourceGatheringMetrics.TokenDistribution[tokenType] += amount
}

// RecordStationVisit records a player visiting a station
func (as *AnalyticsService) RecordStationVisit(playerID string, station string) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.analytics == nil {
		return
	}

	playerAnalytics, exists := as.analytics.PlayerAnalytics[playerID]
	if !exists {
		return
	}

	playerAnalytics.StationPreferences[station]++
	as.analytics.ResourceGatheringMetrics.StationDistribution[station]++
}

// RecordSegmentCompletion records individual puzzle completion
func (as *AnalyticsService) RecordSegmentCompletion(playerID string, solveTime float64) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.analytics == nil {
		return
	}

	playerAnalytics, exists := as.analytics.PlayerAnalytics[playerID]
	if !exists {
		return
	}

	playerAnalytics.IndividualSolveTime = solveTime

	// Update puzzle metrics
	puzzleMetrics := as.analytics.PuzzleAssemblyMetrics

	// Update average solve time
	completedCount := 0
	totalTime := 0.0
	for _, pa := range as.analytics.PlayerAnalytics {
		if pa.IndividualSolveTime > 0 {
			completedCount++
			totalTime += pa.IndividualSolveTime
		}
	}

	if completedCount > 0 {
		puzzleMetrics.AverageSolveTime = totalTime / float64(completedCount)
	}

	// Update fastest/slowest
	if puzzleMetrics.FastestCompletion == 0 || solveTime < puzzleMetrics.FastestCompletion {
		puzzleMetrics.FastestCompletion = solveTime
	}
	if solveTime > puzzleMetrics.SlowestCompletion {
		puzzleMetrics.SlowestCompletion = solveTime
	}
}

// RecordFragmentMove records a fragment move
func (as *AnalyticsService) RecordFragmentMove(playerID string, successful bool) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.analytics == nil {
		return
	}

	playerAnalytics, exists := as.analytics.PlayerAnalytics[playerID]
	if !exists {
		return
	}

	playerAnalytics.FragmentMoves++
	if successful {
		playerAnalytics.SuccessfulMoves++
	}

	// Update puzzle metrics
	as.analytics.PuzzleAssemblyMetrics.TotalMoves++
	if successful {
		as.analytics.PuzzleAssemblyMetrics.SuccessfulMoves++
	}
}

// RecordRecommendation records a recommendation
func (as *AnalyticsService) RecordRecommendation(fromPlayerID, toPlayerID string, accepted bool) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.analytics == nil {
		return
	}

	// Update sender stats
	if fromAnalytics, exists := as.analytics.PlayerAnalytics[fromPlayerID]; exists {
		fromAnalytics.RecommendationsSent++
	}

	// Update receiver stats
	if toAnalytics, exists := as.analytics.PlayerAnalytics[toPlayerID]; exists {
		toAnalytics.RecommendationsReceived++
		if accepted {
			toAnalytics.RecommendationsAccepted++
		}
	}

	// Update puzzle metrics
	as.analytics.PuzzleAssemblyMetrics.TotalRecommendations++
	if accepted {
		as.analytics.PuzzleAssemblyMetrics.AcceptedRecommendations++
	}
}

// FinalizeGame calculates final analytics when game ends
func (as *AnalyticsService) FinalizeGame(game *models.Game, players map[string]*models.Player, success bool) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.analytics == nil {
		return
	}

	// Set game completion info
	as.analytics.EndTime = time.Now()
	as.analytics.Duration = as.analytics.EndTime.Sub(as.analytics.StartTime).Seconds()
	as.analytics.Difficulty = game.Difficulty

	// Update team analytics
	as.analytics.TeamAnalytics.GameSuccess = success
	as.analytics.TeamAnalytics.TotalPlayers = len(players)
	as.analytics.TeamAnalytics.GameTime = as.analytics.Duration
	as.analytics.TeamAnalytics.PuzzleCompletionTime = game.CompletionTime

	// Calculate resource gathering metrics
	as.calculateResourceMetrics(game)

	// Calculate puzzle metrics
	as.calculatePuzzleMetrics(game)

	// Calculate player scores and rankings
	as.calculatePlayerScores(game, players, success)

	// Determine achievements
	as.determineAchievements(game, players)

	// Calculate team metrics
	as.calculateTeamMetrics(game, players)

	// Finalize game analytics
	as.analytics.Finalize(game, players)

	log.Printf("Analytics finalized for game %s - Success: %v", game.ID, success)
}

// calculateResourceMetrics calculates resource gathering phase metrics
func (as *AnalyticsService) calculateResourceMetrics(game *models.Game) {
	metrics := as.analytics.ResourceGatheringMetrics

	metrics.TotalRounds = config.ResourceGatheringRounds

	// Calculate overall accuracy
	totalQuestions := 0
	correctAnswers := 0
	for _, playerAnalytics := range as.analytics.PlayerAnalytics {
		totalQuestions += playerAnalytics.TotalQuestions
		correctAnswers += playerAnalytics.CorrectAnswers
	}

	if totalQuestions > 0 {
		metrics.OverallAccuracy = float64(correctAnswers) / float64(totalQuestions)
		as.analytics.TeamAnalytics.OverallAccuracy = metrics.OverallAccuracy
	}

	// Set token distribution from game state
	metrics.TokenDistribution[models.TokenAnchor] = game.TeamTokens.AnchorTokens
	metrics.TokenDistribution[models.TokenChronos] = game.TeamTokens.ChronosTokens
	metrics.TokenDistribution[models.TokenGuide] = game.TeamTokens.GuideTokens
	metrics.TokenDistribution[models.TokenClarity] = game.TeamTokens.ClarityTokens

	as.analytics.TeamAnalytics.TotalTokensCollected =
		game.TeamTokens.AnchorTokens + game.TeamTokens.ChronosTokens +
			game.TeamTokens.GuideTokens + game.TeamTokens.ClarityTokens
}

// calculatePuzzleMetrics calculates puzzle phase metrics
func (as *AnalyticsService) calculatePuzzleMetrics(game *models.Game) {
	metrics := as.analytics.PuzzleAssemblyMetrics

	metrics.TotalTime = float64(game.GetTotalPuzzleTime())
	metrics.CompletionTime = game.CompletionTime

	if metrics.TotalTime > 0 {
		metrics.TimeUtilization = metrics.CompletionTime / metrics.TotalTime
	}

	// Calculate move accuracy
	if metrics.TotalMoves > 0 {
		metrics.MoveAccuracy = float64(metrics.SuccessfulMoves) / float64(metrics.TotalMoves)
	}

	// Calculate recommendation acceptance rate
	if metrics.TotalRecommendations > 0 {
		metrics.RecommendationAcceptanceRate =
			float64(metrics.AcceptedRecommendations) / float64(metrics.TotalRecommendations)
	}

	// Set pre-solved pieces used
	metrics.PreSolvedPiecesUsed = game.GetPreSolvedPieces() * len(as.analytics.PlayerAnalytics)
}

// calculatePlayerScores calculates individual player scores
func (as *AnalyticsService) calculatePlayerScores(game *models.Game, players map[string]*models.Player, success bool) {
	// Calculate scores for each player
	for playerID, playerAnalytics := range as.analytics.PlayerAnalytics {
		// Set player role if available
		if player, exists := players[playerID]; exists {
			playerAnalytics.Role = player.Role
		}

		// Calculate accuracy
		if playerAnalytics.TotalQuestions > 0 {
			playerAnalytics.Accuracy = float64(playerAnalytics.CorrectAnswers) /
				float64(playerAnalytics.TotalQuestions)
		}

		// Calculate specialty accuracy
		if playerAnalytics.SpecialtyQuestions > 0 {
			playerAnalytics.SpecialtyAccuracy = float64(playerAnalytics.SpecialtyCorrect) /
				float64(playerAnalytics.SpecialtyQuestions)
		}

		// Calculate move accuracy
		if playerAnalytics.FragmentMoves > 0 {
			playerAnalytics.MoveAccuracy = float64(playerAnalytics.SuccessfulMoves) /
				float64(playerAnalytics.FragmentMoves)
		}

		// Calculate score
		playerAnalytics.CalculateScore(success, game.CompletionTime)
	}

	// Rank players by score
	as.rankPlayers()
}

// rankPlayers assigns ranks to players based on scores
func (as *AnalyticsService) rankPlayers() {
	// Create sorted list of player IDs by score
	type playerScore struct {
		ID    string
		Score int
	}

	scores := []playerScore{}
	for id, analytics := range as.analytics.PlayerAnalytics {
		scores = append(scores, playerScore{ID: id, Score: analytics.TotalScore})
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Assign ranks
	for rank, ps := range scores {
		as.analytics.PlayerAnalytics[ps.ID].Rank = rank + 1
	}

	// Also rank by individual solve time
	solveTimes := []playerScore{}
	for id, analytics := range as.analytics.PlayerAnalytics {
		if analytics.IndividualSolveTime > 0 {
			solveTimes = append(solveTimes, playerScore{
				ID:    id,
				Score: int(analytics.IndividualSolveTime * 1000), // Convert to ms for int comparison
			})
		}
	}

	// Sort by solve time (ascending - faster is better)
	for i := 0; i < len(solveTimes); i++ {
		for j := i + 1; j < len(solveTimes); j++ {
			if solveTimes[j].Score < solveTimes[i].Score {
				solveTimes[i], solveTimes[j] = solveTimes[j], solveTimes[i]
			}
		}
	}

	// Assign individual ranks
	for rank, st := range solveTimes {
		as.analytics.PlayerAnalytics[st.ID].IndividualRank = rank + 1
	}
}

// determineAchievements determines achievements for players and team
func (as *AnalyticsService) determineAchievements(game *models.Game, players map[string]*models.Player) {
	// Individual achievements
	for _, playerAnalytics := range as.analytics.PlayerAnalytics {
		playerAnalytics.DetermineAchievements()
	}

	// Team achievements
	as.analytics.TeamAnalytics.DetermineTeamAchievements(game)
}

// calculateTeamMetrics calculates team-wide metrics
func (as *AnalyticsService) calculateTeamMetrics(game *models.Game, players map[string]*models.Player) {
	teamAnalytics := as.analytics.TeamAnalytics

	// Calculate total team score
	totalScore := 0
	for _, playerAnalytics := range as.analytics.PlayerAnalytics {
		totalScore += playerAnalytics.TotalScore
	}
	teamAnalytics.TotalScore = totalScore

	// Set threshold achievements
	teamAnalytics.ThresholdAchievements[models.TokenAnchor] = game.TeamTokens.GetThreshold(models.TokenAnchor)
	teamAnalytics.ThresholdAchievements[models.TokenChronos] = game.TeamTokens.GetThreshold(models.TokenChronos)
	teamAnalytics.ThresholdAchievements[models.TokenGuide] = game.TeamTokens.GetThreshold(models.TokenGuide)
	teamAnalytics.ThresholdAchievements[models.TokenClarity] = game.TeamTokens.GetThreshold(models.TokenClarity)

	// Calculate collaboration efficiency
	if as.analytics.PuzzleAssemblyMetrics.TotalMoves > 0 {
		teamAnalytics.CollaborationEfficiency = as.analytics.PuzzleAssemblyMetrics.MoveAccuracy
	}

	// Determine notable players
	as.determineNotableStats()
}

// determineNotableStats determines notable player statistics
func (as *AnalyticsService) determineNotableStats() {
	var (
		fastestAnswerer  string
		fastestTime      float64 = 999999
		mostTokens       string
		maxTokens        int
		bestCollaborator string
		maxCollab        float64
		puzzleMVP        string
		maxMoves         int
	)

	for _, analytics := range as.analytics.PlayerAnalytics {
		// Fastest answerer
		if analytics.AverageResponseTime > 0 && analytics.AverageResponseTime < fastestTime {
			fastestTime = analytics.AverageResponseTime
			fastestAnswerer = analytics.PlayerName
		}

		// Most tokens
		if analytics.TotalTokens > maxTokens {
			maxTokens = analytics.TotalTokens
			mostTokens = analytics.PlayerName
		}

		// Best collaborator
		if analytics.CollaborationScore > maxCollab {
			maxCollab = analytics.CollaborationScore
			bestCollaborator = analytics.PlayerName
		}

		// Puzzle MVP
		if analytics.SuccessfulMoves > maxMoves {
			maxMoves = analytics.SuccessfulMoves
			puzzleMVP = analytics.PlayerName
		}
	}

	as.analytics.TeamAnalytics.FastestAnswerer = fastestAnswerer
	as.analytics.TeamAnalytics.MostTokens = mostTokens
	as.analytics.TeamAnalytics.BestCollaborator = bestCollaborator
	as.analytics.TeamAnalytics.PuzzleMVP = puzzleMVP
}

// GetFullAnalytics returns complete game analytics
func (as *AnalyticsService) GetFullAnalytics() *models.GameAnalytics {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return as.analytics
}

// Reset resets analytics for a new game
func (as *AnalyticsService) Reset() {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.analytics = nil
	log.Println("Analytics reset for new game")
}
