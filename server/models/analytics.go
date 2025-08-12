package models

import (
	"canvas-conundrum/constants"
	"time"
)

// PlayerAnalytics represents individual player performance metrics
type PlayerAnalytics struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Role       Role   `json:"role"`

	// Trivia Performance
	TotalQuestions      int                        `json:"totalQuestions"`
	CorrectAnswers      int                        `json:"correctAnswers"`
	Accuracy            float64                    `json:"accuracy"`
	AccuracyByCategory  map[TriviaCategory]float64 `json:"accuracyByCategory"`
	SpecialtyQuestions  int                        `json:"specialtyQuestions"`
	SpecialtyCorrect    int                        `json:"specialtyCorrect"`
	SpecialtyAccuracy   float64                    `json:"specialtyAccuracy"`
	SpecialtyBonus      int                        `json:"specialtyBonus"`
	AverageResponseTime float64                    `json:"averageResponseTime"`

	// Token Collection
	TokensEarned map[TokenType]int `json:"tokensEarned"`
	TotalTokens  int               `json:"totalTokens"`

	// Puzzle Performance
	IndividualSolveTime float64 `json:"individualSolveTime"`
	IndividualRank      int     `json:"individualRank"`
	FragmentMoves       int     `json:"fragmentMoves"`
	SuccessfulMoves     int     `json:"successfulMoves"`
	MoveAccuracy        float64 `json:"moveAccuracy"`

	// Collaboration
	RecommendationsSent     int     `json:"recommendationsSent"`
	RecommendationsReceived int     `json:"recommendationsReceived"`
	RecommendationsAccepted int     `json:"recommendationsAccepted"`
	CollaborationScore      float64 `json:"collaborationScore"`

	// Scoring
	TriviaPoints       int `json:"triviaPoints"`
	PuzzlePoints       int `json:"puzzlePoints"`
	CollaborationBonus int `json:"collaborationBonus"`
	SpeedBonus         int `json:"speedBonus"`
	TotalScore         int `json:"totalScore"`
	Rank               int `json:"rank"`

	// Achievements
	Achievements []string `json:"achievements"`

	// Station preferences (how many times at each station)
	StationPreferences map[string]int `json:"stationPreferences"`
}

// CalculateScore calculates the player's total score
func (pa *PlayerAnalytics) CalculateScore(gameSuccess bool, puzzleTime float64) {
	// Trivia points
	pa.TriviaPoints = pa.CorrectAnswers * constants.PointsPerCorrectAnswer

	// Specialty bonus
	pa.SpecialtyBonus = pa.SpecialtyCorrect * constants.SpecialtyBonusPoints * constants.PointsPerCorrectAnswer

	// Puzzle points
	if gameSuccess {
		pa.PuzzlePoints = constants.CompletionBonus

		// Speed bonus (faster solve time = more points)
		if pa.IndividualSolveTime > 0 {
			speedRatio := 1.0 - (pa.IndividualSolveTime / puzzleTime)
			if speedRatio > 0 {
				pa.SpeedBonus = int(float64(constants.MaxSpeedBonus) * speedRatio)
			}
		}
	}

	// Movement points
	pa.PuzzlePoints += pa.SuccessfulMoves * constants.PointsPerSuccessfulMove

	// Collaboration bonus
	pa.CollaborationBonus = pa.RecommendationsSent * constants.PointsPerRecommendationSent
	pa.CollaborationBonus += pa.RecommendationsAccepted * constants.PointsPerRecommendationAccepted

	// Total score
	pa.TotalScore = pa.TriviaPoints + pa.SpecialtyBonus + pa.PuzzlePoints +
		pa.CollaborationBonus + pa.SpeedBonus

	// Calculate collaboration score (0-1)
	if pa.RecommendationsSent > 0 || pa.RecommendationsReceived > 0 {
		totalInteractions := float64(pa.RecommendationsSent + pa.RecommendationsReceived)
		acceptanceRate := float64(pa.RecommendationsAccepted) / totalInteractions
		pa.CollaborationScore = acceptanceRate
	}
}

// DetermineAchievements determines which achievements the player earned
func (pa *PlayerAnalytics) DetermineAchievements() {
	pa.Achievements = []string{}

	// Trivia achievements
	if pa.Accuracy >= 0.9 {
		pa.Achievements = append(pa.Achievements, "Trivia Master")
	} else if pa.Accuracy >= 0.75 {
		pa.Achievements = append(pa.Achievements, "Quiz Whiz")
	}

	// Speed achievements
	if pa.IndividualRank == 1 {
		pa.Achievements = append(pa.Achievements, "Speed Demon")
	}

	// Collaboration achievements
	if pa.CollaborationScore >= 0.8 {
		pa.Achievements = append(pa.Achievements, "Team Player")
	}

	if pa.RecommendationsAccepted >= 5 {
		pa.Achievements = append(pa.Achievements, "Strategic Thinker")
	}

	// Perfect game
	if pa.Accuracy == 1.0 && pa.MoveAccuracy == 1.0 {
		pa.Achievements = append(pa.Achievements, "Perfectionist")
	}
}

// TeamAnalytics represents team-wide performance metrics
type TeamAnalytics struct {
	GameSuccess  bool    `json:"gameSuccess"`
	TotalScore   int     `json:"totalScore"`
	TotalPlayers int     `json:"totalPlayers"`
	GameTime     float64 `json:"gameTime"`

	// Team Performance
	OverallAccuracy         float64           `json:"overallAccuracy"`
	TotalTokensCollected    int               `json:"totalTokensCollected"`
	ThresholdAchievements   map[TokenType]int `json:"thresholdAchievements"`
	PuzzleCompletionTime    float64           `json:"puzzleCompletionTime"`
	CollaborationEfficiency float64           `json:"collaborationEfficiency"`

	// Notable Stats
	FastestAnswerer  string `json:"fastestAnswerer"`
	MostTokens       string `json:"mostTokens"`
	BestCollaborator string `json:"bestCollaborator"`
	PuzzleMVP        string `json:"puzzleMVP"`

	// Team Achievements
	TeamAchievements []string `json:"teamAchievements"`
}

// GameAnalytics represents complete game analytics
type GameAnalytics struct {
	GameID     string         `json:"gameId"`
	StartTime  time.Time      `json:"startTime"`
	EndTime    time.Time      `json:"endTime"`
	Duration   float64        `json:"duration"`
	Difficulty DifficultyMode `json:"difficulty"`

	// Player Analytics
	PlayerAnalytics map[string]*PlayerAnalytics `json:"playerAnalytics"`

	// Team Analytics
	TeamAnalytics *TeamAnalytics `json:"teamAnalytics"`

	// Phase Metrics
	ResourceGatheringMetrics *ResourceGatheringMetrics `json:"resourceGatheringMetrics"`
	PuzzleAssemblyMetrics    *PuzzleAssemblyMetrics    `json:"puzzleAssemblyMetrics"`

	// Category Performance
	CategoryPerformance map[TriviaCategory]*CategoryStats `json:"categoryPerformance"`

	// Recommendations
	RecommendationsForImprovement []string `json:"recommendationsForImprovement"`
}

// ResourceGatheringMetrics tracks resource phase performance
type ResourceGatheringMetrics struct {
	TotalRounds         int               `json:"totalRounds"`
	QuestionsAnswered   int               `json:"questionsAnswered"`
	OverallAccuracy     float64           `json:"overallAccuracy"`
	TokenDistribution   map[TokenType]int `json:"tokenDistribution"`
	AverageResponseTime float64           `json:"averageResponseTime"`
	StationDistribution map[string]int    `json:"stationDistribution"`
}

// PuzzleAssemblyMetrics tracks puzzle phase performance
type PuzzleAssemblyMetrics struct {
	TotalTime       float64 `json:"totalTime"`
	CompletionTime  float64 `json:"completionTime"`
	TimeUtilization float64 `json:"timeUtilization"`

	// Individual Phase
	AverageSolveTime    float64 `json:"averageSolveTime"`
	FastestCompletion   float64 `json:"fastestCompletion"`
	SlowestCompletion   float64 `json:"slowestCompletion"`
	PreSolvedPiecesUsed int     `json:"preSolvedPiecesUsed"`

	// Collaborative Phase
	TotalMoves                   int     `json:"totalMoves"`
	SuccessfulMoves              int     `json:"successfulMoves"`
	MoveAccuracy                 float64 `json:"moveAccuracy"`
	TotalRecommendations         int     `json:"totalRecommendations"`
	AcceptedRecommendations      int     `json:"acceptedRecommendations"`
	RecommendationAcceptanceRate float64 `json:"recommendationAcceptanceRate"`
}

// CategoryStats tracks performance for a trivia category
type CategoryStats struct {
	QuestionsAsked int     `json:"questionsAsked"`
	CorrectAnswers int     `json:"correctAnswers"`
	Accuracy       float64 `json:"accuracy"`
}

// NewGameAnalytics creates a new game analytics instance
func NewGameAnalytics(gameID string) *GameAnalytics {
	return &GameAnalytics{
		GameID:          gameID,
		StartTime:       time.Now(),
		PlayerAnalytics: make(map[string]*PlayerAnalytics),
		TeamAnalytics: &TeamAnalytics{
			ThresholdAchievements: make(map[TokenType]int),
		},
		CategoryPerformance: make(map[TriviaCategory]*CategoryStats),
		ResourceGatheringMetrics: &ResourceGatheringMetrics{
			TokenDistribution:   make(map[TokenType]int),
			StationDistribution: make(map[string]int),
		},
		PuzzleAssemblyMetrics: &PuzzleAssemblyMetrics{},
	}
}

// Finalize calculates final analytics when game ends
func (ga *GameAnalytics) Finalize(game *Game, players map[string]*Player) {
	ga.EndTime = time.Now()
	ga.Duration = ga.EndTime.Sub(ga.StartTime).Seconds()
	ga.Difficulty = game.Difficulty

	// Set team success
	ga.TeamAnalytics.GameSuccess = game.PuzzleSuccess
	ga.TeamAnalytics.TotalPlayers = len(players)
	ga.TeamAnalytics.GameTime = ga.Duration

	// Calculate team metrics
	ga.calculateTeamMetrics(game, players)

	// Generate recommendations
	ga.generateRecommendations()
}

// calculateTeamMetrics calculates team-wide performance metrics
func (ga *GameAnalytics) calculateTeamMetrics(game *Game, players map[string]*Player) {
	// Token thresholds
	ga.TeamAnalytics.ThresholdAchievements[TokenAnchor] = game.TeamTokens.GetThreshold(TokenAnchor)
	ga.TeamAnalytics.ThresholdAchievements[TokenChronos] = game.TeamTokens.GetThreshold(TokenChronos)
	ga.TeamAnalytics.ThresholdAchievements[TokenGuide] = game.TeamTokens.GetThreshold(TokenGuide)
	ga.TeamAnalytics.ThresholdAchievements[TokenClarity] = game.TeamTokens.GetThreshold(TokenClarity)

	// Puzzle completion time
	if game.PuzzleSuccess {
		ga.TeamAnalytics.PuzzleCompletionTime = game.CompletionTime
	}

	// Team achievements
	ga.TeamAnalytics.DetermineTeamAchievements(game)
}

// DetermineTeamAchievements determines team-level achievements
func (ta *TeamAnalytics) DetermineTeamAchievements(game *Game) {
	achievements := []string{}

	if game.PuzzleSuccess {
		achievements = append(achievements, "Puzzle Champions")

		// Check for perfect collaboration
		// Simplified for now - always add if successful
		achievements = append(achievements, "Perfect Collaboration")

		// Check for token mastery
		totalThresholds := 0
		for _, level := range ta.ThresholdAchievements {
			totalThresholds += level
		}
		if totalThresholds >= 12 {
			achievements = append(achievements, "Token Masters")
		}
	}

	ta.TeamAchievements = achievements
}

// generateRecommendations generates improvement recommendations
func (ga *GameAnalytics) generateRecommendations() {
	recommendations := []string{}

	// Check trivia performance
	if ga.ResourceGatheringMetrics.OverallAccuracy < 0.6 {
		recommendations = append(recommendations, "Consider reviewing trivia categories with low accuracy")
	}

	// Check collaboration
	if ga.PuzzleAssemblyMetrics.RecommendationAcceptanceRate < 0.5 {
		recommendations = append(recommendations, "Improve communication during puzzle assembly")
	}

	// Check time utilization
	if ga.PuzzleAssemblyMetrics.TimeUtilization < 0.7 {
		recommendations = append(recommendations, "Work on solving puzzles more quickly")
	}

	// Positive feedback
	if ga.TeamAnalytics.GameSuccess {
		recommendations = append(recommendations, "Great teamwork and coordination!")
	}

	ga.RecommendationsForImprovement = recommendations
}
