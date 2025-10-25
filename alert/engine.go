package alert

import (
	"log/slog"
	"sync"
)

type AlertingEngine struct {
	mu sync.RWMutex

	rules  map[string]AlertRule
	logger *slog.Logger
}

func NewAlertingEngine(logger *slog.Logger) *AlertingEngine {
	return &AlertingEngine{
		rules:  make(map[string]AlertRule),
		logger: logger,
	}
}

func (e *AlertingEngine) AddRule(rule AlertRule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules[rule.ID] = rule
}

func (e *AlertingEngine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.rules, ruleID)
}

func (e *AlertingEngine) Evaluate(metric Metric) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if rule.MetricName != metric.Name {
			e.logger.Debug("Skipping rule for different metric", "ruleID", rule.ID, "ruleName", rule.Name, "expectedMetric", rule.MetricName, "actualMetric", metric.Name)
			continue
		}

		if !rule.Enabled {
			e.logger.Info("Skipping disabled rule", "ruleID", rule.ID, "ruleName", rule.Name)
			continue
		}

		if rule.Condition(metric) {
			e.logger.Info("Rule condition met, executing action", "ruleID", rule.ID, "ruleName", rule.Name)
			if err := rule.Action(metric, rule); err != nil {
				e.logger.Error("Failed to execute rule action", "ruleID", rule.ID, "ruleName", rule.Name, "error", err)
			}
		} else {
			e.logger.Info("Rule condition not met", "ruleID", rule.ID, "ruleName", rule.Name)
		}
	}
}
