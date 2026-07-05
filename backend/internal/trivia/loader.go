// Package trivia loads Open Trivia DB question pools from
// TRIVIA_DIR/{category}/{difficulty}.json and serves them with pool cycling
// (game-design.md § Question Management).
package trivia

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
)

const (
	Easy   = "easy"
	Medium = "medium"
	Hard   = "hard"
)

// Difficulties in ascending order.
var Difficulties = []string{Easy, Medium, Hard}

// Bump returns the next-harder difficulty, capped at hard. Specialty
// questions are one difficulty above the game's base difficulty.
func Bump(difficulty string) string {
	switch difficulty {
	case Easy:
		return Medium
	default:
		return Hard
	}
}

// Question is one decoded trivia question. Category and Difficulty come
// from the pool's directory/file name; the raw file's own category,
// difficulty, and type fields are ignored per the spec.
type Question struct {
	Category   string
	Difficulty string
	Index      int // position within the pool file
	Text       string
	Correct    string
	Incorrect  []string
}

// ID renders the wire question identifier: {category}_{difficulty}_{index}.
func (q Question) ID() string {
	return fmt.Sprintf("%s_%s_%d", q.Category, q.Difficulty, q.Index)
}

type poolKey struct{ category, difficulty string }

// Bank holds every loaded pool, immutable after Load.
type Bank struct {
	categories []string
	pools      map[poolKey][]Question
}

// openTDBFile is the raw Open Trivia DB export shape.
type openTDBFile struct {
	ResponseCode int `json:"response_code"`
	Results      []struct {
		Question         string   `json:"question"`
		CorrectAnswer    string   `json:"correct_answer"`
		IncorrectAnswers []string `json:"incorrect_answers"`
	} `json:"results"`
}

// Load walks dir treating each subdirectory as a category, requiring all
// three difficulty pools per category, each non-empty. HTML entities are
// decoded here so the rest of the server only sees clean text.
func Load(dir string) (*Bank, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read trivia dir: %w", err)
	}

	b := &Bank{pools: map[poolKey][]Question{}}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		category := e.Name()
		for _, difficulty := range Difficulties {
			path := filepath.Join(dir, category, difficulty+".json")
			pool, err := loadPool(path, category, difficulty)
			if err != nil {
				return nil, err
			}
			b.pools[poolKey{category, difficulty}] = pool
		}
		b.categories = append(b.categories, category)
	}
	if len(b.categories) == 0 {
		return nil, fmt.Errorf("no trivia categories found under %s", dir)
	}
	sort.Strings(b.categories)
	return b, nil
}

func loadPool(path, category, difficulty string) ([]Question, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("trivia pool: %w", err)
	}
	var file openTDBFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse trivia pool %s: %w", path, err)
	}
	if len(file.Results) == 0 {
		return nil, fmt.Errorf("trivia pool %s is empty", path)
	}

	pool := make([]Question, len(file.Results))
	for i, r := range file.Results {
		incorrect := make([]string, len(r.IncorrectAnswers))
		for j, a := range r.IncorrectAnswers {
			incorrect[j] = html.UnescapeString(a)
		}
		pool[i] = Question{
			Category:   category,
			Difficulty: difficulty,
			Index:      i,
			Text:       html.UnescapeString(r.Question),
			Correct:    html.UnescapeString(r.CorrectAnswer),
			Incorrect:  incorrect,
		}
	}
	return pool, nil
}

// Categories returns the sorted category keys.
func (b *Bank) Categories() []string { return b.categories }

// PoolSize returns the number of questions in one pool (0 if absent).
func (b *Bank) PoolSize(category, difficulty string) int {
	return len(b.pools[poolKey{category, difficulty}])
}
