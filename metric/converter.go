package metric

import "github.com/prometheus/client_golang/prometheus"

type Converter interface {
	Name() string
	Match(name string) bool
	Convert(any) (prometheus.Collector, error)
}
