package alert

import "math"

const DefaultFloatTolerance = 0.0001

type Metric struct {
	Name        string
	Description string

	Value     float64
	Unit      string
	Timestamp int64
}

type RuleCondition func(metric Metric) bool

type AlertRule struct {
	ID         string
	Name       string
	MetricName string
	Enabled    bool

	Condition RuleCondition
	Action    RuleAction
}

// common condition builders
func ThresholdAbove(threshold float64) RuleCondition {
	return func(metric Metric) bool {
		return metric.Value > threshold
	}
}

func ThresholdBelow(threshold float64) RuleCondition {
	return func(metric Metric) bool {
		return metric.Value < threshold
	}
}

func ThresholdBetween(min, max float64) RuleCondition {
	return func(metric Metric) bool {
		return metric.Value >= min && metric.Value <= max
	}
}

// ThresholdEquals creates a condition that checks for equality with tolerance
func ThresholdEquals(target float64) RuleCondition {
	return func(metric Metric) bool {
		return FloatEquals(metric.Value, target, DefaultFloatTolerance)
	}
}

func FloatEquals(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}
