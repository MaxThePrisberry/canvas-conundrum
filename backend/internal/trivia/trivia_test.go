package trivia

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T) *Bank {
	t.Helper()
	b, err := Load(filepath.Join("testdata", "trivia"))
	if err != nil {
		t.Fatalf("Load fixtures: %v", err)
	}
	return b
}

func TestLoadFixtures(t *testing.T) {
	b := loadFixture(t)

	want := []string{"alpha", "beta"}
	got := b.Categories()
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Categories() = %v, want %v", got, want)
	}
	for _, cat := range want {
		for _, diff := range Difficulties {
			if n := b.PoolSize(cat, diff); n != 2 {
				t.Errorf("PoolSize(%s,%s) = %d, want 2", cat, diff, n)
			}
		}
	}
}

func TestHTMLEntitiesDecoded(t *testing.T) {
	b := loadFixture(t)
	q := b.pools[poolKey{"alpha", "easy"}][1]

	if want := `alpha easy question two: "quoted" & entity-laden?`; q.Text != want {
		t.Errorf("question text = %q, want %q", q.Text, want)
	}
	if q.Correct != "Café" {
		t.Errorf("correct answer = %q, want Café", q.Correct)
	}
	if q.Incorrect[0] != "A <tag>" {
		t.Errorf("incorrect[0] = %q, want A <tag>", q.Incorrect[0])
	}
}

func TestQuestionID(t *testing.T) {
	q := Question{Category: "science", Difficulty: "medium", Index: 7}
	if got := q.ID(); got != "science_medium_7" {
		t.Errorf("ID() = %q", got)
	}
}

func TestBump(t *testing.T) {
	cases := map[string]string{Easy: Medium, Medium: Hard, Hard: Hard}
	for in, want := range cases {
		if got := Bump(in); got != want {
			t.Errorf("Bump(%s) = %s, want %s", in, got, want)
		}
	}
}

// TestDeckCycles proves each full cycle deals every question exactly once
// and no question repeats back-to-back across cycle boundaries.
func TestDeckCycles(t *testing.T) {
	b := loadFixture(t)
	d := NewDeck(b)
	rng := rand.New(rand.NewPCG(1, 2))

	var prev Question
	for cycle := 0; cycle < 50; cycle++ {
		seen := map[int]bool{}
		for i := 0; i < 2; i++ {
			q := d.Next("alpha", Medium, rng)
			if seen[q.Index] {
				t.Fatalf("cycle %d dealt index %d twice", cycle, q.Index)
			}
			seen[q.Index] = true
			if cycle+i > 0 && q.Index == prev.Index {
				t.Fatalf("immediate repeat of index %d at cycle %d", q.Index, cycle)
			}
			prev = q
		}
	}
}

// TestLoadRealTriviaDir parses the repo's committed trivia content, proving
// the loader handles the real Open Trivia DB exports.
func TestLoadRealTriviaDir(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "trivia")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("repo trivia dir not present: %v", err)
	}

	b, err := Load(dir)
	if err != nil {
		t.Fatalf("Load real trivia: %v", err)
	}
	if len(b.Categories()) != 6 {
		t.Errorf("real trivia has %d categories, want 6", len(b.Categories()))
	}
	for _, cat := range b.Categories() {
		for _, diff := range Difficulties {
			pool := b.pools[poolKey{cat, diff}]
			if len(pool) == 0 {
				t.Errorf("pool %s/%s empty", cat, diff)
			}
			for _, q := range pool {
				if strings.Contains(q.Text, "&amp;") || strings.Contains(q.Correct, "&quot;") {
					t.Errorf("undecoded entity in %s: %q", q.ID(), q.Text)
				}
			}
		}
	}
}

func TestLoadRejectsMissingDifficulty(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "solo"), 0o755); err != nil {
		t.Fatal(err)
	}
	pool := `{"response_code":0,"results":[{"question":"q","correct_answer":"a","incorrect_answers":["b"]}]}`
	if err := os.WriteFile(filepath.Join(dir, "solo", "easy.json"), []byte(pool), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("Load accepted a category missing medium/hard pools")
	}
}

func TestLoadRejectsEmptyDir(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Error("Load accepted an empty trivia dir")
	}
}
