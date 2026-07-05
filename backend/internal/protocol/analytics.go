package protocol

import "fmt"

// ── Client → Server ────────────────────────────────────────────────────────

// ResetGame is ANALYTICS_TO_SERVER_RESET_GAME (valid only during analytics).
type ResetGame struct {
	ConfirmReset bool `json:"confirmReset"`
}

func (r ResetGame) Validate() error {
	if !r.ConfirmReset {
		return fmt.Errorf("confirmReset must be true")
	}
	return nil
}

// ── Server → Client ────────────────────────────────────────────────────────

// TokenCollection is one player's station-attributed token earnings.
type TokenCollection struct {
	AnchorTokens  int `json:"anchorTokens"`
	ChronosTokens int `json:"chronosTokens"`
	GuideTokens   int `json:"guideTokens"`
	ClarityTokens int `json:"clarityTokens"`
	TotalTokens   int `json:"totalTokens"`
}

// PersonalSpecialtyPerformance is the specialty block in the personal report.
type PersonalSpecialtyPerformance struct {
	SpecialtyQuestions int     `json:"specialtyQuestions"`
	SpecialtyCorrect   int     `json:"specialtyCorrect"`
	SpecialtyAccuracy  float64 `json:"specialtyAccuracy"`
	BonusTokens        int     `json:"bonusTokens"`
}

// TriviaPerformance is the resource-phase block in the personal report.
type TriviaPerformance struct {
	TotalQuestions       int                          `json:"totalQuestions"`
	CorrectAnswers       int                          `json:"correctAnswers"`
	Accuracy             float64                      `json:"accuracy"`
	AccuracyByCategory   map[string]float64           `json:"accuracyByCategory"`
	SpecialtyPerformance PersonalSpecialtyPerformance `json:"specialtyPerformance"`
	AverageResponseTime  float64                      `json:"averageResponseTime"`
}

// PuzzleSolvingMetrics is the assembly block in the personal report.
type PuzzleSolvingMetrics struct {
	IndividualSolveTime     float64 `json:"individualSolveTime"`
	IndividualRank          int     `json:"individualRank"`
	FragmentMoves           int     `json:"fragmentMoves"`
	SuccessfulMoves         int     `json:"successfulMoves"`
	MoveAccuracy            float64 `json:"moveAccuracy"`
	RecommendationsSent     int     `json:"recommendationsSent"`
	RecommendationsReceived int     `json:"recommendationsReceived"`
	RecommendationsAccepted int     `json:"recommendationsAccepted"`
}

// ScoreBreakdown mirrors the Scoring Algorithm terms; RecommendationPoints
// combines the sent and accepted terms.
type ScoreBreakdown struct {
	TriviaPoints         int `json:"triviaPoints"`
	SpecialtyPoints      int `json:"specialtyPoints"`
	CompletionBonus      int `json:"completionBonus"`
	MovePoints           int `json:"movePoints"`
	RecommendationPoints int `json:"recommendationPoints"`
	TotalScore           int `json:"totalScore"`
}

// PersonalReport is ANALYTICS_TO_PLAYER_PERSONAL_REPORT.
type PersonalReport struct {
	PlayerID             string               `json:"playerId"`
	PlayerName           string               `json:"playerName"`
	GameSuccess          bool                 `json:"gameSuccess"`
	PersonalScore        int                  `json:"personalScore"`
	Rank                 int                  `json:"rank"`
	TotalPlayers         int                  `json:"totalPlayers"`
	TokenCollection      TokenCollection      `json:"tokenCollection"`
	TriviaPerformance    TriviaPerformance    `json:"triviaPerformance"`
	PuzzleSolvingMetrics PuzzleSolvingMetrics `json:"puzzleSolvingMetrics"`
	ScoreBreakdown       ScoreBreakdown       `json:"scoreBreakdown"`
}

// LeaderboardEntry is one row of the team summary leaderboard.
type LeaderboardEntry struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	TotalScore int    `json:"totalScore"`
	Rank       int    `json:"rank"`
	Role       string `json:"role"`
}

// SummaryTeamPerformance is the team block in ANALYTICS_TO_CLIENT_TEAM_SUMMARY.
type SummaryTeamPerformance struct {
	OverallAccuracy       float64      `json:"overallAccuracy"`
	TotalTokensCollected  int          `json:"totalTokensCollected"`
	ThresholdAchievements ThresholdSet `json:"thresholdAchievements"`
	PuzzleCompletionTime  float64      `json:"puzzleCompletionTime"`
}

// TeamSummary is ANALYTICS_TO_CLIENT_TEAM_SUMMARY. The leaderboard uses
// competition ranking (1, 2, 2, 4), alphabetical by name within ties.
type TeamSummary struct {
	GameSuccess     bool                   `json:"gameSuccess"`
	TotalScore      int                    `json:"totalScore"`
	TotalPlayers    int                    `json:"totalPlayers"`
	TotalGameTime   float64                `json:"totalGameTime"`
	TeamPerformance SummaryTeamPerformance `json:"teamPerformance"`
	Leaderboard     []LeaderboardEntry     `json:"leaderboard"`
}

// OverallPerformance is the score block in ANALYTICS_TO_HOST_COMPLETE_REPORT.
type OverallPerformance struct {
	TotalScore     int     `json:"totalScore"`
	AverageScore   float64 `json:"averageScore"`
	CompletionRate float64 `json:"completionRate"`
}

// HostResourcePlayerPerformance is one player's resource row in the host
// complete report.
type HostResourcePlayerPerformance struct {
	QuestionsAnswered    int                  `json:"questionsAnswered"`
	CorrectAnswers       int                  `json:"correctAnswers"`
	Accuracy             float64              `json:"accuracy"`
	TokensEarned         int                  `json:"tokensEarned"`
	AverageResponseTime  float64              `json:"averageResponseTime"`
	SpecialtyPerformance SpecialtyPerformance `json:"specialtyPerformance"`
	StationPreferences   map[string]int       `json:"stationPreferences"`
}

// ResourceGatheringAnalytics is the resource block of the host report.
type ResourceGatheringAnalytics struct {
	TotalRounds       int                                      `json:"totalRounds"`
	QuestionsAnswered int                                      `json:"questionsAnswered"`
	OverallAccuracy   float64                                  `json:"overallAccuracy"`
	TokenDistribution TeamTokens                               `json:"tokenDistribution"`
	PlayerPerformance map[string]HostResourcePlayerPerformance `json:"playerPerformance"`
}

// IndividualPhaseMetrics summarizes 2A in the host report.
type IndividualPhaseMetrics struct {
	AverageSolveTime    float64 `json:"averageSolveTime"`
	FastestCompletion   float64 `json:"fastestCompletion"`
	SlowestCompletion   float64 `json:"slowestCompletion"`
	PreSolvedPiecesUsed int     `json:"preSolvedPiecesUsed"`
}

// CollaborativePhaseMetrics summarizes 2B in the host report.
type CollaborativePhaseMetrics struct {
	TotalMoves                   int     `json:"totalMoves"`
	SuccessfulMoves              int     `json:"successfulMoves"`
	MoveAccuracy                 float64 `json:"moveAccuracy"`
	TotalRecommendations         int     `json:"totalRecommendations"`
	AcceptedRecommendations      int     `json:"acceptedRecommendations"`
	RecommendationAcceptanceRate float64 `json:"recommendationAcceptanceRate"`
}

// HostPlayerContribution is one player's assembly row in the host report.
type HostPlayerContribution struct {
	IndividualSolveTime     float64 `json:"individualSolveTime"`
	FragmentMoves           int     `json:"fragmentMoves"`
	SuccessfulMoves         int     `json:"successfulMoves"`
	RecommendationsSent     int     `json:"recommendationsSent"`
	RecommendationsReceived int     `json:"recommendationsReceived"`
	RecommendationsAccepted int     `json:"recommendationsAccepted"`
}

// PuzzleAssemblyAnalytics is the assembly block of the host report.
type PuzzleAssemblyAnalytics struct {
	TotalTime                 float64                           `json:"totalTime"`
	CompletionTime            float64                           `json:"completionTime"`
	TimeUtilization           float64                           `json:"timeUtilization"`
	IndividualPhaseMetrics    IndividualPhaseMetrics            `json:"individualPhaseMetrics"`
	CollaborativePhaseMetrics CollaborativePhaseMetrics         `json:"collaborativePhaseMetrics"`
	PlayerContributions       map[string]HostPlayerContribution `json:"playerContributions"`
}

// CategoryPerformance is one category row of the host report.
type CategoryPerformance struct {
	QuestionsAsked int     `json:"questionsAsked"`
	CorrectAnswers int     `json:"correctAnswers"`
	Accuracy       float64 `json:"accuracy"`
}

// TimelineAnalysis carries phase durations in seconds; they sum to
// totalGameTime.
type TimelineAnalysis struct {
	SetupPhase       float64 `json:"setupPhase"`
	ResourcePhase    float64 `json:"resourcePhase"`
	PreparationPhase float64 `json:"preparationPhase"`
	PuzzlePhase      float64 `json:"puzzlePhase"`
}

// HostCompleteReport is ANALYTICS_TO_HOST_COMPLETE_REPORT.
type HostCompleteReport struct {
	GameSuccess                bool                           `json:"gameSuccess"`
	TotalGameTime              float64                        `json:"totalGameTime"`
	TotalPlayers               int                            `json:"totalPlayers"`
	DifficultyMode             string                         `json:"difficultyMode"`
	OverallPerformance         OverallPerformance             `json:"overallPerformance"`
	ResourceGatheringAnalytics ResourceGatheringAnalytics     `json:"resourceGatheringAnalytics"`
	PuzzleAssemblyAnalytics    PuzzleAssemblyAnalytics        `json:"puzzleAssemblyAnalytics"`
	CategoryPerformance        map[string]CategoryPerformance `json:"categoryPerformance"`
	TimelineAnalysis           TimelineAnalysis               `json:"timelineAnalysis"`
}

// GameReset is ANALYTICS_TO_CLIENT_GAME_RESET (all participants).
type GameReset struct {
	Reason                string `json:"reason"`
	ReconnectRequired     bool   `json:"reconnectRequired"`
	ReconnectInstructions string `json:"reconnectInstructions"`
	NewGameAvailable      bool   `json:"newGameAvailable"`
}
