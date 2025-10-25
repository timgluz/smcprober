package alert

import "log/slog"

type RuleAction func(metric Metric, rule AlertRule) error

// common action builders
func LogAction(logger *slog.Logger) RuleAction {
	return func(metric Metric, rule AlertRule) error {
		logger.Info("Alert triggered", "ruleID", rule.ID, "ruleName", rule.Name, "metric", metric.Name, "value", metric.Value, "unit", metric.Unit)
		return nil
	}
}

func NoOpAction() RuleAction {
	return func(metric Metric, rule AlertRule) error {
		return nil
	}
}

func MultiAction(actions ...RuleAction) RuleAction {
	return func(metric Metric, rule AlertRule) error {
		for _, action := range actions {
			if err := action(metric, rule); err != nil {
				return err
			}
		}
		return nil
	}
}
