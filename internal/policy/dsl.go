package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Policy represents a decision policy
type Policy struct {
	ID          string            `yaml:"id"`
	Version     string            `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"` // autoscale, ratelimit, circuitbreaker
	Rules       []Rule            `yaml:"rules"`
	Defaults    map[string]string `yaml:"defaults"`
}

// Rule represents a single rule in a policy
type Rule struct {
	ID       string      `yaml:"id"`
	Name     string      `yaml:"name"`
	Priority int         `yaml:"priority"`
	When     Condition   `yaml:"when"`
	Action   Action      `yaml:"action"`
	Cooldown string      `yaml:"cooldown,omitempty"`
}

// Condition represents a rule condition
type Condition struct {
	All  []Condition `yaml:"all,omitempty"`
	Any  []Condition `yaml:"any,omitempty"`
	Not  *Condition  `yaml:"not,omitempty"`
	Fact string      `yaml:"fact,omitempty"`
	Op   string      `yaml:"op,omitempty"`     // >, <, >=, <=, ==, !=
	Value interface{} `yaml:"value,omitempty"`
}

// Action represents the action to take when rule matches
type Action struct {
	Type    string                 `yaml:"type"`    // scale_up, scale_down, throttle, etc
	Target  string                 `yaml:"target,omitempty"`
	Params  map[string]interface{} `yaml:"params,omitempty"`
	Cost    float64                `yaml:"cost,omitempty"`
	Risk    float64                `yaml:"risk,omitempty"`
}

// LoadPolicy loads a policy from a YAML file
func LoadPolicy(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy: %w", err)
	}

	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	return &policy, nil
}

// Validate validates the policy structure
func (p *Policy) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("policy id is required")
	}
	if p.Version == "" {
		return fmt.Errorf("policy version is required")
	}
	if len(p.Rules) == 0 {
		return fmt.Errorf("policy must have at least one rule")
	}

	// Check for duplicate rule IDs
	ruleIDs := make(map[string]bool)
	for _, rule := range p.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rule id is required")
		}
		if ruleIDs[rule.ID] {
			return fmt.Errorf("duplicate rule id: %s", rule.ID)
		}
		ruleIDs[rule.ID] = true

		if err := rule.Validate(); err != nil {
			return fmt.Errorf("invalid rule %s: %w", rule.ID, err)
		}
	}

	return nil
}

// Validate validates a rule
func (r *Rule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if r.Action.Type == "" {
		return fmt.Errorf("rule action type is required")
	}
	return nil
}

// GetRuleByID finds a rule by its ID
func (p *Policy) GetRuleByID(id string) *Rule {
	for i := range p.Rules {
		if p.Rules[i].ID == id {
			return &p.Rules[i]
		}
	}
	return nil
}
