package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads and parses the config file. Unknown keys are rejected so a
// typo'd tunable fails loudly at startup instead of silently using a zero
// value.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}
