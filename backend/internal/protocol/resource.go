package protocol

import "fmt"

// ── Client → Server ────────────────────────────────────────────────────────

// LocationVerified is RESOURCE_TO_SERVER_LOCATION_VERIFIED — the player
// scanned a station QR code. PreviousLocation is optional (absent on the
// first scan).
type LocationVerified struct {
	StationHash      string  `json:"stationHash"`
	PreviousLocation *string `json:"previousLocation"`
	ScanTimestamp    string  `json:"scanTimestamp"`
}

func (l LocationVerified) Validate() error {
	return validateText("stationHash", l.StationHash, 1, MaxStationHashLen)
}

// TriviaAnswer is RESOURCE_TO_SERVER_TRIVIA_ANSWER. Resubmission before the
// deadline overwrites; late answers are silently ignored.
type TriviaAnswer struct {
	QuestionID  string  `json:"questionId"`
	AnswerIndex int     `json:"answerIndex"`
	TimeElapsed float64 `json:"timeElapsed"`
}

func (a TriviaAnswer) Validate() error {
	if a.QuestionID == "" {
		return fmt.Errorf("questionId is required")
	}
	if a.AnswerIndex < 0 {
		return fmt.Errorf("answerIndex must be >= 0")
	}
	return nil
}

// ── Server → Client ────────────────────────────────────────────────────────

// ThresholdSet carries per-token-type threshold values or counts.
type ThresholdSet struct {
	Anchor  int `json:"anchor"`
	Chronos int `json:"chronos"`
	Guide   int `json:"guide"`
	Clarity int `json:"clarity"`
}

// DifficultySettings echoes the difficulty-mode modifiers.
type DifficultySettings struct {
	Mode                 string  `json:"mode"`
	SpecialtyProbability float64 `json:"specialtyProbability"`
	TimeMultiplier       float64 `json:"timeMultiplier"`
	ThresholdMultiplier  float64 `json:"thresholdMultiplier"`
}

// ResourcePhaseStart is RESOURCE_TO_CLIENT_PHASE_START. The server waits one
// roundDuration after sending it before Round 1's questions.
type ResourcePhaseStart struct {
	Phase              string             `json:"phase"`
	TotalRounds        int                `json:"totalRounds"`
	RoundDuration      float64            `json:"roundDuration"`
	AnswerTime         float64            `json:"answerTime"`
	GraceTime          float64            `json:"graceTime"`
	TokenThresholds    ThresholdSet       `json:"tokenThresholds"`
	DifficultySettings DifficultySettings `json:"difficultySettings"`
}

// MonitoringDashboard is the host's live view in RESOURCE_TO_HOST_PHASE_START.
type MonitoringDashboard struct {
	TotalRounds        int            `json:"totalRounds"`
	CurrentRound       int            `json:"currentRound"`
	RoundDuration      float64        `json:"roundDuration"`
	PlayerDistribution map[string]int `json:"playerDistribution"`
}

// HostResourcePhaseStart is RESOURCE_TO_HOST_PHASE_START.
type HostResourcePhaseStart struct {
	Phase               string              `json:"phase"`
	MonitoringDashboard MonitoringDashboard `json:"monitoringDashboard"`
}

// LocationConfirmed is RESOURCE_TO_PLAYER_LOCATION_CONFIRMED.
type LocationConfirmed struct {
	NewLocation string `json:"newLocation"`
}

// TriviaQuestion is RESOURCE_TO_PLAYER_TRIVIA_QUESTION.
type TriviaQuestion struct {
	QuestionID     string   `json:"questionId"`
	QuestionText   string   `json:"questionText"`
	Category       string   `json:"category"`
	Difficulty     string   `json:"difficulty"`
	IsSpecialty    bool     `json:"isSpecialty"`
	Options        []string `json:"options"`
	RoundNumber    int      `json:"roundNumber"`
	TotalRounds    int      `json:"totalRounds"`
	AnswerDeadline string   `json:"answerDeadline"`
}

// AnswerBonuses decomposes a token award for display.
type AnswerBonuses struct {
	RoleBonus            bool `json:"roleBonus"`
	RoleBonusTokens      int  `json:"roleBonusTokens"`
	SpecialtyBonus       bool `json:"specialtyBonus"`
	SpecialtyBonusTokens int  `json:"specialtyBonusTokens"`
}

// AnswerResult is RESOURCE_TO_PLAYER_ANSWER_RESULT, sent when the answer
// window closes. SelectedAnswer is null when the player never answered.
type AnswerResult struct {
	QuestionID          string        `json:"questionId"`
	Correct             bool          `json:"correct"`
	SelectedAnswer      *string       `json:"selectedAnswer"`
	CorrectAnswer       string        `json:"correctAnswer"`
	TokensEarned        int           `json:"tokensEarned"`
	BaseTokens          int           `json:"baseTokens"`
	Bonuses             AnswerBonuses `json:"bonuses"`
	CurrentLocation     string        `json:"currentLocation"`
	NextTriviaTimestamp string        `json:"nextTriviaTimestamp"`
}

// TeamPerformance is the aggregate block in RESOURCE_TO_CLIENT_TEAM_PROGRESS.
type TeamPerformance struct {
	AverageAccuracy    float64 `json:"averageAccuracy"`
	RoundTimeRemaining float64 `json:"roundTimeRemaining"`
}

// TeamProgress is RESOURCE_TO_CLIENT_TEAM_PROGRESS. TotalQuestions is the
// full-phase delivery count (players × rounds); accuracy counts one slot per
// player per elapsed round, unanswered/undelivered = incorrect.
type TeamProgress struct {
	CurrentRound      int             `json:"currentRound"`
	TotalRounds       int             `json:"totalRounds"`
	QuestionsAnswered int             `json:"questionsAnswered"`
	TotalQuestions    int             `json:"totalQuestions"`
	TeamTokens        TeamTokens      `json:"teamTokens"`
	CurrentThresholds ThresholdSet    `json:"currentThresholds"`
	TeamPerformance   TeamPerformance `json:"teamPerformance"`
}

// RoundResults aggregates one round in RESOURCE_TO_HOST_ROUND_ANALYTICS.
type RoundResults struct {
	QuestionsDelivered  int     `json:"questionsDelivered"`
	AnswersReceived     int     `json:"answersReceived"`
	CorrectAnswers      int     `json:"correctAnswers"`
	AverageResponseTime float64 `json:"averageResponseTime"`
	TokensAwarded       int     `json:"tokensAwarded"`
}

// PlayerRoundPerformance is one player's row in RESOURCE_TO_HOST_ROUND_ANALYTICS.
type PlayerRoundPerformance struct {
	Location        string  `json:"location"`
	AnswerCorrect   bool    `json:"answerCorrect"`
	ResponseTime    float64 `json:"responseTime"`
	TokensEarned    int     `json:"tokensEarned"`
	RunningAccuracy float64 `json:"runningAccuracy"`
}

// RoundAnalytics is RESOURCE_TO_HOST_ROUND_ANALYTICS.
type RoundAnalytics struct {
	CurrentRound        int                               `json:"currentRound"`
	TotalRounds         int                               `json:"totalRounds"`
	RoundResults        RoundResults                      `json:"roundResults"`
	PlayerPerformance   map[string]PlayerRoundPerformance `json:"playerPerformance"`
	StationDistribution map[string]int                    `json:"stationDistribution"`
	TeamTokens          TeamTokens                        `json:"teamTokens"`
}

// BonusEffects translates threshold achievements into puzzle-phase effects.
type BonusEffects struct {
	AnchorPreSolved        int     `json:"anchorPreSolved"`
	ChronosTimeBonus       float64 `json:"chronosTimeBonus"`
	GuideHighlightCount    int     `json:"guideHighlightCount"`
	ClarityPreviewDuration float64 `json:"clarityPreviewDuration"`
}

// ResourcePhaseComplete is RESOURCE_TO_CLIENT_PHASE_COMPLETE.
type ResourcePhaseComplete struct {
	Phase                 string       `json:"phase"`
	NextPhase             string       `json:"nextPhase"`
	FinalTokenTotals      TeamTokens   `json:"finalTokenTotals"`
	ThresholdAchievements ThresholdSet `json:"thresholdAchievements"`
	BonusEffects          BonusEffects `json:"bonusEffects"`
}

// SpecialtyPerformance summarizes a player's specialty questions.
type SpecialtyPerformance struct {
	QuestionsReceived int `json:"questionsReceived"`
	CorrectAnswers    int `json:"correctAnswers"`
	BonusTokens       int `json:"bonusTokens"`
}

// ResourcePlayerAnalytics is one player's block in RESOURCE_TO_HOST_PHASE_COMPLETE.
type ResourcePlayerAnalytics struct {
	QuestionsAnswered    int                  `json:"questionsAnswered"`
	CorrectAnswers       int                  `json:"correctAnswers"`
	Accuracy             float64              `json:"accuracy"`
	TokensEarned         int                  `json:"tokensEarned"`
	SpecialtyPerformance SpecialtyPerformance `json:"specialtyPerformance"`
}

// HostTeamPerformance is the team block in RESOURCE_TO_HOST_PHASE_COMPLETE.
type HostTeamPerformance struct {
	OverallAccuracy     float64 `json:"overallAccuracy"`
	TotalTokensEarned   int     `json:"totalTokensEarned"`
	AverageResponseTime float64 `json:"averageResponseTime"`
}

// HostResourcePhaseComplete is RESOURCE_TO_HOST_PHASE_COMPLETE.
type HostResourcePhaseComplete struct {
	Phase                  string                             `json:"phase"`
	TotalQuestionsAnswered int                                `json:"totalQuestionsAnswered"`
	TeamPerformance        HostTeamPerformance                `json:"teamPerformance"`
	FinalTokenDistribution TeamTokens                         `json:"finalTokenDistribution"`
	PlayerAnalytics        map[string]ResourcePlayerAnalytics `json:"playerAnalytics"`
	ReadyForPuzzlePhase    bool                               `json:"readyForPuzzlePhase"`
}
