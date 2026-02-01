package policy

// Policy represents a decision policy
type Policy struct {
	ID          string            `yaml:"id" json:"id"`
	Version     string            `yaml:"version" json:"version"`
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description" json:"description"`
	Type        string            `yaml:"type" json:"type"`
	Rules       []Rule            `yaml:"rules" json:"rules"`
	Defaults    map[string]string `yaml:"defaults" json:"defaults"`
}

// Rule represents a single rule in a policy
type Rule struct {
	ID       string    `yaml:"id" json:"id"`
	Name     string    `yaml:"name" json:"name"`
	Priority int       `yaml:"priority" json:"priority"`
	When     Condition `yaml:"when" json:"when"`
	Action   Action    `yaml:"action" json:"action"`
	Cooldown string    `yaml:"cooldown,omitempty" json:"cooldown,omitempty"`
}

// Condition represents a rule condition
type Condition struct {
	All  []Condition `yaml:"all,omitempty" json:"all,omitempty"`
	Any  []Condition `yaml:"any,omitempty" json:"any,omitempty"`
	Not  *Condition  `yaml:"not,omitempty" json:"not,omitempty"`
	Fact string      `yaml:"fact,omitempty" json:"fact,omitempty"`
	Op   string      `yaml:"op,omitempty" json:"op,omitempty"`
	Value interface{} `yaml:"value,omitempty" json:"value,omitempty"`
}

// Action represents the action to take when rule matches
type Action struct {
	Type   string                 `yaml:"type" json:"type"`
	Target string                 `yaml:"target,omitempty" json:"target,omitempty"`
	Params map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"`
	Cost   float64                `yaml:"cost,omitempty" json:"cost,omitempty"`
	Risk   float64                `yaml:"risk,omitempty" json:"risk,omitempty"`
}

// Validate validates the policy structure
func (p *Policy) Validate() error {
	if p.ID == "" {
		return &PolicyValidationError{Field: "id", Message: "policy ID is required"}
	}
	if p.Version == "" {
		return &PolicyValidationError{Field: "version", Message: "policy version is required"}
	}
	if len(p.Rules) == 0 {
		return &PolicyValidationError{Field: "rules", Message: "policy must have at least one rule"}
	}

	ruleIDs := make(map[string]bool)
	for _, rule := range p.Rules {
		if rule.ID == "" {
			return &PolicyValidationError{Field: "rule.id", Message: "rule ID is required"}
		}
		if ruleIDs[rule.ID] {
			return &PolicyValidationError{Field: "rule.id", Message: "duplicate rule ID: " + rule.ID}
		}
		ruleIDs[rule.ID] = true

		if err := validateRule(&rule); err != nil {
			return err
		}
	}

	return nil
}

func validateRule(r *Rule) error {
	if r.Name == "" {
		return &PolicyValidationError{Field: "rule.name", Message: "rule name is required"}
	}
	if r.Action.Type == "" {
		return &PolicyValidationError{Field: "rule.action.type", Message: "rule action type is required"}
	}
	return nil
}

// PolicyValidationError represents a policy validation error
type PolicyValidationError struct {
	Field   string
	Message string
}

func (e *PolicyValidationError) Error() string {
	return e.Field + ": " + e.Message
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
