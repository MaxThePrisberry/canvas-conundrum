package game

import (
	"math"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/trivia"
)

// Resource-gathering phase (game-design.md § Phase 1): a fixed number of
// rounds, each one answer window + one grace window, preceded by a single
// silent round-length wait so players can reach and scan their first QR
// station. Host disconnection has no effect here, so none of these timers
// are pausable.
const (
	timerResourceWait   = "resource.wait"
	timerResourceAnswer = "resource.answer"
	timerResourceGrace  = "resource.grace"
)

type resourceState struct {
	round          int // 0 during the initial wait, 1-based during rounds
	playersAtStart int
	lastAnalytics  *protocol.RoundAnalytics
}

func (e *Engine) roundDuration() time.Duration {
	return e.cfg.TriviaAnswerTime.Duration() + e.cfg.TriviaGraceTime.Duration()
}

// enterResourceGathering transitions setup → resource_gathering.
func (e *Engine) enterResourceGathering() {
	// Players still disconnected when the game starts are removed: setup
	// disconnects are removals (state preserved only for setup-phase
	// reconnection), and the game's player set is fixed from here on.
	for id, p := range e.players {
		if !p.Connected {
			delete(e.players, id)
		}
	}

	e.phase = protocol.PhaseResourceGathering
	e.resource = resourceState{playersAtStart: len(e.players)}

	e.broadcastPlayers(protocol.ResourceToClientPhaseStart, e.buildResourcePhaseStart())
	e.sendHost(protocol.ResourceToHostPhaseStart, e.buildHostResourcePhaseStart())

	// One silent round-length wait before Round 1's questions.
	e.timers.Schedule(timerResourceWait, e.roundDuration(), false)
}

func (e *Engine) handleResourceTimer(name string) {
	switch name {
	case timerResourceWait:
		e.startRound(1)
	case timerResourceAnswer:
		e.closeAnswerWindow()
	case timerResourceGrace:
		if e.resource.round >= e.cfg.ResourceGatheringRounds {
			e.enterPuzzlePreparation()
		} else {
			e.startRound(e.resource.round + 1)
		}
	}
}

// ── Round lifecycle ────────────────────────────────────────────────────────

func (e *Engine) startRound(round int) {
	e.resource.round = round
	deadline := time.Now().Add(e.cfg.TriviaAnswerTime.Duration())

	for _, p := range e.players {
		p.Question = nil
		if !p.Connected {
			continue // no questions while disconnected; the slot counts as incorrect
		}
		q := e.drawQuestion(p)
		q.Deadline = deadline
		p.Question = q

		p.Stats.QuestionsDelivered++
		if q.IsSpecialty {
			p.Stats.SpecialtyDelivered++
		}

		p.send(protocol.ResourceToPlayerTriviaQuestion, e.buildTriviaQuestion(q, round))
	}

	e.timers.Schedule(timerResourceAnswer, e.cfg.TriviaAnswerTime.Duration(), false)
}

// drawQuestion rolls the per-player specialty probability and deals from the
// matching pool. Specialty questions come from one of the player's specialty
// categories at one difficulty above the game's base (capped at hard);
// regular questions from a uniformly random category at base difficulty.
func (e *Engine) drawQuestion(p *Player) *ActiveQuestion {
	base := e.cfg.DifficultyMode
	isSpecialty := len(p.Specialties) > 0 && e.rng.Float64() < e.cfg.SpecialtyProbability()

	var category, difficulty string
	if isSpecialty {
		category = p.Specialties[e.rng.IntN(len(p.Specialties))]
		difficulty = trivia.Bump(base)
	} else {
		cats := e.bank.Categories()
		category = cats[e.rng.IntN(len(cats))]
		difficulty = base
	}

	q := e.deck.Next(category, difficulty, e.rng)

	options := make([]string, 0, len(q.Incorrect)+1)
	options = append(options, q.Correct)
	options = append(options, q.Incorrect...)
	e.rng.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	correctIndex := 0
	for i, o := range options {
		if o == q.Correct {
			correctIndex = i
			break
		}
	}

	return &ActiveQuestion{
		Q:            q,
		Options:      options,
		CorrectIndex: correctIndex,
		IsSpecialty:  isSpecialty,
	}
}

func (e *Engine) buildTriviaQuestion(q *ActiveQuestion, round int) protocol.TriviaQuestion {
	return protocol.TriviaQuestion{
		QuestionID:     q.Q.ID(),
		QuestionText:   q.Q.Text,
		Category:       q.Q.Category,
		Difficulty:     q.Q.Difficulty,
		IsSpecialty:    q.IsSpecialty,
		Options:        q.Options,
		RoundNumber:    round,
		TotalRounds:    e.cfg.ResourceGatheringRounds,
		AnswerDeadline: protocol.Timestamp(q.Deadline),
	}
}

// closeAnswerWindow marks every delivered question, awards tokens to the
// station on record at this moment, and emits the per-round events.
func (e *Engine) closeAnswerWindow() {
	round := e.resource.round
	nextTrivia := protocol.Timestamp(time.Now().Add(e.cfg.TriviaGraceTime.Duration()))

	results := protocol.RoundResults{}
	var responseTimeSum float64
	performance := map[string]protocol.PlayerRoundPerformance{}

	for id, p := range e.players {
		perf := protocol.PlayerRoundPerformance{Location: stationOrUnknown(p)}

		if q := p.Question; q != nil && !q.Closed {
			q.Closed = true
			results.QuestionsDelivered++

			answered := q.Answer != nil
			correct := answered && q.Answer.Index == q.CorrectIndex

			var earned int
			var bonuses protocol.AnswerBonuses
			baseTokens := 0
			if correct {
				earned, bonuses = e.awardTokens(p, q.IsSpecialty)
				if earned > 0 {
					baseTokens = e.cfg.BaseTokensPerCorrectAnswer
				}
				e.addTokens(p.Station, earned)
			}

			if answered {
				results.AnswersReceived++
				responseTimeSum += q.Answer.TimeElapsed
				p.Stats.QuestionsAnswered++
				p.Stats.TotalResponseTime += q.Answer.TimeElapsed
				perf.ResponseTime = q.Answer.TimeElapsed
			}
			if correct {
				results.CorrectAnswers++
				p.Stats.CorrectAnswers++
				if q.IsSpecialty {
					p.Stats.SpecialtyCorrect++
					p.Stats.SpecialtyBonusTokens += bonuses.SpecialtyBonusTokens
				}
			}
			p.Stats.countCategory(q.Q.Category, correct)
			p.Stats.TokensEarned += earned
			results.TokensAwarded += earned

			perf.AnswerCorrect = correct
			perf.TokensEarned = earned

			var selected *string
			if answered {
				s := q.Options[min(q.Answer.Index, len(q.Options)-1)]
				selected = &s
			}
			p.send(protocol.ResourceToPlayerAnswerResult, protocol.AnswerResult{
				QuestionID:          q.Q.ID(),
				Correct:             correct,
				SelectedAnswer:      selected,
				CorrectAnswer:       q.Q.Correct,
				TokensEarned:        earned,
				BaseTokens:          baseTokens,
				Bonuses:             bonuses,
				CurrentLocation:     stationOrUnknown(p),
				NextTriviaTimestamp: nextTrivia,
			})
		}

		perf.RunningAccuracy = round2(float64(p.Stats.CorrectAnswers) / float64(round))
		performance[id] = perf
	}

	if results.AnswersReceived > 0 {
		results.AverageResponseTime = round2(responseTimeSum / float64(results.AnswersReceived))
	}

	analytics := protocol.RoundAnalytics{
		CurrentRound:        round,
		TotalRounds:         e.cfg.ResourceGatheringRounds,
		RoundResults:        results,
		PlayerPerformance:   performance,
		StationDistribution: e.stationDistribution(),
		TeamTokens:          e.tokens,
	}
	e.resource.lastAnalytics = &analytics

	e.broadcastPlayers(protocol.ResourceToClientTeamProgress, e.buildTeamProgress())
	e.sendHost(protocol.ResourceToHostRoundAnalytics, analytics)

	e.timers.Schedule(timerResourceGrace, e.cfg.TriviaGraceTime.Duration(), false)
}

// ── Player events ──────────────────────────────────────────────────────────

func (e *Engine) handleLocationVerified(p *Player, payload protocol.LocationVerified) {
	if e.phase != protocol.PhaseResourceGathering {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"station scans are only accepted during resource gathering", "")
		return
	}

	station, ok := e.stationForHash(payload.StationHash)
	if !ok {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrInvalidStationHash,
			"QR-code hash did not match any configured station", "")
		return
	}

	p.Station = station
	if p.Stats.StationVisits == nil {
		p.Stats.StationVisits = map[string]int{}
	}
	p.Stats.StationVisits[station]++

	p.send(protocol.ResourceToPlayerLocationConfirmed, protocol.LocationConfirmed{NewLocation: station})
}

// stationForHash maps a scanned QR payload to its station. Hashes are
// server-side only and never sent to clients.
func (e *Engine) stationForHash(hash string) (string, bool) {
	switch hash {
	case e.cfg.StationHashes.Anchor:
		return stationAnchor, true
	case e.cfg.StationHashes.Chronos:
		return stationChronos, true
	case e.cfg.StationHashes.Guide:
		return stationGuide, true
	case e.cfg.StationHashes.Clarity:
		return stationClarity, true
	}
	return "", false
}

func (e *Engine) handleTriviaAnswer(p *Player, payload protocol.TriviaAnswer) {
	if e.phase != protocol.PhaseResourceGathering {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"trivia answers are only accepted during resource gathering", "")
		return
	}
	q := p.Question
	// Late, duplicate-round, or mismatched answers are silently ignored per
	// the spec; the last answer before the deadline counts.
	if q == nil || q.Closed || payload.QuestionID != q.Q.ID() {
		return
	}
	if payload.AnswerIndex >= len(q.Options) {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrMalformedPayload,
			"answerIndex is out of range", "")
		return
	}
	elapsed := math.Min(math.Max(payload.TimeElapsed, 0), e.cfg.TriviaAnswerTime.Sec())
	q.Answer = &SubmittedAnswer{Index: payload.AnswerIndex, TimeElapsed: elapsed}
}

// ── Payload builders ───────────────────────────────────────────────────────

func stationOrUnknown(p *Player) string {
	if p.Station == "" {
		return stationUnknown
	}
	return p.Station
}

func (e *Engine) stationDistribution() map[string]int {
	dist := map[string]int{stationUnknown: 0}
	for _, s := range stationNames {
		dist[s] = 0
	}
	for _, p := range e.players {
		dist[stationOrUnknown(p)]++
	}
	return dist
}

func (e *Engine) buildResourcePhaseStart() protocol.ResourcePhaseStart {
	return protocol.ResourcePhaseStart{
		Phase:         protocol.PhaseResourceGathering,
		TotalRounds:   e.cfg.ResourceGatheringRounds,
		RoundDuration: e.roundDuration().Seconds(),
		AnswerTime:    e.cfg.TriviaAnswerTime.Sec(),
		GraceTime:     e.cfg.TriviaGraceTime.Sec(),
		TokenThresholds: protocol.ThresholdSet{
			Anchor:  e.cfg.AnchorTokenThreshold,
			Chronos: e.cfg.ChronosTokenThreshold,
			Guide:   e.cfg.GuideTokenThreshold,
			Clarity: e.cfg.ClarityTokenThreshold,
		},
		DifficultySettings: protocol.DifficultySettings{
			Mode:                 e.cfg.DifficultyMode,
			SpecialtyProbability: e.cfg.SpecialtyProbability(),
			TimeMultiplier:       e.cfg.TimeMultiplier(),
			ThresholdMultiplier:  e.cfg.ThresholdMultiplier(),
		},
	}
}

func (e *Engine) buildHostResourcePhaseStart() protocol.HostResourcePhaseStart {
	return protocol.HostResourcePhaseStart{
		Phase: protocol.PhaseResourceGathering,
		MonitoringDashboard: protocol.MonitoringDashboard{
			TotalRounds:        e.cfg.ResourceGatheringRounds,
			CurrentRound:       e.resource.round,
			RoundDuration:      e.roundDuration().Seconds(),
			PlayerDistribution: e.stationDistribution(),
		},
	}
}

func (e *Engine) buildTeamProgress() protocol.TeamProgress {
	answered, correct := 0, 0
	for _, p := range e.players {
		answered += p.Stats.QuestionsAnswered
		correct += p.Stats.CorrectAnswers
	}

	// One accuracy slot per player per elapsed round: unanswered and
	// undelivered (disconnected) both count as incorrect.
	accuracy := 0.0
	if slots := e.resource.playersAtStart * e.resource.round; slots > 0 {
		accuracy = round2(float64(correct) / float64(slots))
	}

	return protocol.TeamProgress{
		CurrentRound:      e.resource.round,
		TotalRounds:       e.cfg.ResourceGatheringRounds,
		QuestionsAnswered: answered,
		TotalQuestions:    e.resource.playersAtStart * e.cfg.ResourceGatheringRounds,
		TeamTokens:        e.tokens,
		CurrentThresholds: e.currentThresholds(),
		TeamPerformance: protocol.TeamPerformance{
			AverageAccuracy:    accuracy,
			RoundTimeRemaining: round2(e.resourceTimeRemaining().Seconds()),
		},
	}
}

// resourceTimeRemaining reports time left on whichever phase timer is
// currently pending.
func (e *Engine) resourceTimeRemaining() time.Duration {
	for _, name := range []string{timerResourceAnswer, timerResourceGrace, timerResourceWait} {
		if d := e.timers.Remaining(name); d > 0 {
			return d
		}
	}
	return 0
}

// ── Phase completion ───────────────────────────────────────────────────────

// enterPuzzlePreparation ends resource gathering: emits both PHASE_COMPLETE
// events and hands over to the puzzle-preparation flow (tile generation in
// prep.go).
func (e *Engine) enterPuzzlePreparation() {
	e.phase = protocol.PhasePuzzlePreparation

	thresholds := e.currentThresholds()
	gridSize := e.gridSize()

	e.broadcastPlayers(protocol.ResourceToClientPhaseComplete, protocol.ResourcePhaseComplete{
		Phase:                 protocol.PhaseResourceGathering,
		NextPhase:             protocol.PhasePuzzlePreparation,
		FinalTokenTotals:      e.tokens,
		ThresholdAchievements: thresholds,
		BonusEffects:          e.bonusEffects(thresholds, gridSize),
	})
	e.sendHost(protocol.ResourceToHostPhaseComplete, e.buildHostResourcePhaseComplete())

	e.startPuzzlePreparation()
}

func (e *Engine) buildHostResourcePhaseComplete() protocol.HostResourcePhaseComplete {
	rounds := e.cfg.ResourceGatheringRounds
	slots := e.resource.playersAtStart * rounds

	totalCorrect, totalTokens, totalAnswered := 0, 0, 0
	var responseTimeSum float64
	analytics := map[string]protocol.ResourcePlayerAnalytics{}

	for id, p := range e.players {
		totalCorrect += p.Stats.CorrectAnswers
		totalTokens += p.Stats.TokensEarned
		totalAnswered += p.Stats.QuestionsAnswered
		responseTimeSum += p.Stats.TotalResponseTime

		analytics[id] = protocol.ResourcePlayerAnalytics{
			QuestionsAnswered: rounds,
			CorrectAnswers:    p.Stats.CorrectAnswers,
			Accuracy:          round2(float64(p.Stats.CorrectAnswers) / float64(rounds)),
			TokensEarned:      p.Stats.TokensEarned,
			SpecialtyPerformance: protocol.SpecialtyPerformance{
				QuestionsReceived: p.Stats.SpecialtyDelivered,
				CorrectAnswers:    p.Stats.SpecialtyCorrect,
				BonusTokens:       p.Stats.SpecialtyBonusTokens,
			},
		}
	}

	perf := protocol.HostTeamPerformance{TotalTokensEarned: totalTokens}
	if slots > 0 {
		perf.OverallAccuracy = round2(float64(totalCorrect) / float64(slots))
	}
	if totalAnswered > 0 {
		perf.AverageResponseTime = round2(responseTimeSum / float64(totalAnswered))
	}

	return protocol.HostResourcePhaseComplete{
		Phase:                  protocol.PhaseResourceGathering,
		TotalQuestionsAnswered: slots,
		TeamPerformance:        perf,
		FinalTokenDistribution: e.tokens,
		PlayerAnalytics:        analytics,
		ReadyForPuzzlePhase:    true,
	}
}

// ── Reconnect replay ───────────────────────────────────────────────────────

// replayPlayerResourceState restores a mid-phase reconnector: phase context,
// current team progress, and the in-flight question if the round is open.
func (e *Engine) replayPlayerResourceState(p *Player) {
	p.send(protocol.ResourceToClientPhaseStart, e.buildResourcePhaseStart())
	p.send(protocol.ResourceToClientTeamProgress, e.buildTeamProgress())
	if q := p.Question; q != nil && !q.Closed {
		p.send(protocol.ResourceToPlayerTriviaQuestion, e.buildTriviaQuestion(q, e.resource.round))
	}
}

// replayHostResourceState restores the host dashboard, plus the latest round
// analytics when a round is in progress.
func (e *Engine) replayHostResourceState() {
	e.sendHost(protocol.ResourceToHostPhaseStart, e.buildHostResourcePhaseStart())
	if e.resource.lastAnalytics != nil {
		e.sendHost(protocol.ResourceToHostRoundAnalytics, *e.resource.lastAnalytics)
	}
}

// round2 rounds to two decimals, the display precision used for accuracy
// and response-time fields throughout the spec's examples.
func round2(x float64) float64 {
	return math.Round(x*100) / 100
}
