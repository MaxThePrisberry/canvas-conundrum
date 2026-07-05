package trivia

import "math/rand/v2"

// Deck deals questions from a Bank with automatic pool cycling: each pool is
// dealt in a shuffled order, reshuffled when exhausted, and the reshuffle
// never repeats the previously dealt question first (game-design.md
// § Question Management).
type Deck struct {
	bank  *Bank
	state map[poolKey]*deckState
}

type deckState struct {
	order  []int
	cursor int
	last   int // pool index of the most recently dealt question
}

func NewDeck(bank *Bank) *Deck {
	return &Deck{bank: bank, state: map[poolKey]*deckState{}}
}

// Next deals the next question from the (category, difficulty) pool.
// The pool must exist (Load guarantees all category × difficulty pools).
func (d *Deck) Next(category, difficulty string, rng *rand.Rand) Question {
	key := poolKey{category, difficulty}
	pool := d.bank.pools[key]

	st, ok := d.state[key]
	if !ok {
		st = &deckState{order: shuffled(len(pool), rng), last: -1}
		d.state[key] = st
	}

	if st.cursor >= len(st.order) {
		st.order = shuffled(len(pool), rng)
		// No immediate repeat across the cycle boundary.
		if len(pool) > 1 && st.order[0] == st.last {
			swap := 1 + rng.IntN(len(st.order)-1)
			st.order[0], st.order[swap] = st.order[swap], st.order[0]
		}
		st.cursor = 0
	}

	idx := st.order[st.cursor]
	st.cursor++
	st.last = idx
	return pool[idx]
}

func shuffled(n int, rng *rand.Rand) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	rng.Shuffle(n, func(i, j int) { order[i], order[j] = order[j], order[i] })
	return order
}
