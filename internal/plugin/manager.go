package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Plugin interface that all plugins must implement
type Plugin interface {
	Name() string
	Version() string
	Initialize(config map[string]interface{}, logger *slog.Logger) error
	Shutdown() error
}

// DecisionPlugin is called during decision making
type DecisionPlugin interface {
	Plugin
	BeforeDecision(ctx context.Context, features interface{}) error
	AfterDecision(ctx context.Context, decision interface{}) error
}

// ActionPlugin is called during action execution
type ActionPlugin interface {
	Plugin
	BeforeAction(ctx context.Context, action interface{}) error
	AfterAction(ctx context.Context, result interface{}) error
}

// Manager manages plugins
type Manager struct {
	mu              sync.RWMutex
	plugins         []Plugin
	decisionPlugins []DecisionPlugin
	actionPlugins   []ActionPlugin
	logger          *slog.Logger
}

// NewManager creates a new plugin manager
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		plugins:         make([]Plugin, 0),
		decisionPlugins: make([]DecisionPlugin, 0),
		actionPlugins:   make([]ActionPlugin, 0),
		logger:          logger,
	}
}

// Register registers a plugin
func (m *Manager) Register(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate name
	for _, p := range m.plugins {
		if p.Name() == plugin.Name() {
			return fmt.Errorf("plugin %s already registered", plugin.Name())
		}
	}

	m.plugins = append(m.plugins, plugin)

	// Categorize plugin
	if dp, ok := plugin.(DecisionPlugin); ok {
		m.decisionPlugins = append(m.decisionPlugins, dp)
	}
	if ap, ok := plugin.(ActionPlugin); ok {
		m.actionPlugins = append(m.actionPlugins, ap)
	}

	m.logger.Info("plugin registered", "name", plugin.Name(), "version", plugin.Version())
	return nil
}

// Unregister removes a plugin
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.plugins {
		if p.Name() == name {
			if err := p.Shutdown(); err != nil {
				m.logger.Error("failed to shutdown plugin", "name", name, "error", err)
			}

			// Remove from plugins slice
			m.plugins = append(m.plugins[:i], m.plugins[i+1:]...)

			// Remove from categorized slices
			m.decisionPlugins = filterDecisionPlugins(m.decisionPlugins, name)
			m.actionPlugins = filterActionPlugins(m.actionPlugins, name)

			m.logger.Info("plugin unregistered", "name", name)
			return nil
		}
	}

	return fmt.Errorf("plugin %s not found", name)
}

// ExecuteBeforeDecision executes all decision plugins before decision
func (m *Manager) ExecuteBeforeDecision(ctx context.Context, features interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, plugin := range m.decisionPlugins {
		if err := plugin.BeforeDecision(ctx, features); err != nil {
			m.logger.Error("decision plugin error", "plugin", plugin.Name(), "error", err)
			return fmt.Errorf("plugin %s: %w", plugin.Name(), err)
		}
	}

	return nil
}

// ExecuteAfterDecision executes all decision plugins after decision
func (m *Manager) ExecuteAfterDecision(ctx context.Context, decision interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, plugin := range m.decisionPlugins {
		if err := plugin.AfterDecision(ctx, decision); err != nil {
			m.logger.Error("decision plugin error", "plugin", plugin.Name(), "error", err)
		}
	}

	return nil
}

// ExecuteBeforeAction executes all action plugins before action
func (m *Manager) ExecuteBeforeAction(ctx context.Context, action interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, plugin := range m.actionPlugins {
		if err := plugin.BeforeAction(ctx, action); err != nil {
			m.logger.Error("action plugin error", "plugin", plugin.Name(), "error", err)
			return fmt.Errorf("plugin %s: %w", plugin.Name(), err)
		}
	}

	return nil
}

// ExecuteAfterAction executes all action plugins after action
func (m *Manager) ExecuteAfterAction(ctx context.Context, result interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, plugin := range m.actionPlugins {
		if err := plugin.AfterAction(ctx, result); err != nil {
			m.logger.Error("action plugin error", "plugin", plugin.Name(), "error", err)
		}
	}

	return nil
}

// Shutdown shuts down all plugins
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, plugin := range m.plugins {
		if err := plugin.Shutdown(); err != nil {
			m.logger.Error("failed to shutdown plugin", "name", plugin.Name(), "error", err)
		}
	}

	m.plugins = m.plugins[:0]
	m.decisionPlugins = m.decisionPlugins[:0]
	m.actionPlugins = m.actionPlugins[:0]
}

// ListPlugins returns list of registered plugins
func (m *Manager) ListPlugins() []map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]map[string]string, len(m.plugins))
	for i, p := range m.plugins {
		list[i] = map[string]string{
			"name":    p.Name(),
			"version": p.Version(),
		}
	}

	return list
}

func filterDecisionPlugins(plugins []DecisionPlugin, name string) []DecisionPlugin {
	result := make([]DecisionPlugin, 0, len(plugins))
	for _, p := range plugins {
		if p.Name() != name {
			result = append(result, p)
		}
	}
	return result
}

func filterActionPlugins(plugins []ActionPlugin, name string) []ActionPlugin {
	result := make([]ActionPlugin, 0, len(plugins))
	for _, p := range plugins {
		if p.Name() != name {
			result = append(result, p)
		}
	}
	return result
}
