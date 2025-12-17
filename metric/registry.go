package metric

import (
	"log/slog"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Registry holds all metrics in maps
// and provides methods to get or create them from sensor data using converters
type Registry struct {
	namespace string
	mu        sync.RWMutex

	converters []Converter

	gauges      map[string]prometheus.Gauge
	gaugeVecs   map[string]*prometheus.GaugeVec
	counters    map[string]prometheus.Counter
	counterVecs map[string]*prometheus.CounterVec
	histograms  map[string]prometheus.Histogram

	logger *slog.Logger
}

// NewRegistry creates a new metric registry
func NewRegistry(namespace string, logger *slog.Logger) *Registry {
	return &Registry{
		namespace:   namespace,
		converters:  make([]Converter, 0),
		gauges:      make(map[string]prometheus.Gauge),
		gaugeVecs:   make(map[string]*prometheus.GaugeVec),
		counters:    make(map[string]prometheus.Counter),
		counterVecs: make(map[string]*prometheus.CounterVec),
		histograms:  make(map[string]prometheus.Histogram),
		logger:      logger,
	}
}

func (r *Registry) AddConverters(converters ...Converter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.converters = append(r.converters, converters...)
}

// Convert converts sensor data using registered converters and registers the resulting metric
func (r *Registry) ConvertAndRegister(name string, data any) error {
	converters := r.getMatchingConverters(name)
	if len(converters) == 0 {
		r.logger.Warn("No converters found that match the given name", "name", name)
		return nil
	}

	for _, converter := range converters {
		metric, err := converter.Convert(data)
		if err != nil {
			return err
		}

		if err := prometheus.Register(metric); err != nil {
			// If the metric is already registered, unregister the existing one and register the new one
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				prometheus.Unregister(are.ExistingCollector)
				prometheus.MustRegister(metric)
			} else {
				r.logger.Error("Failed to register metric",
					"converter", converter.Name(), "data", data, "error", err,
				)

				return err
			}
		}
	}

	return nil
}

func (r *Registry) getMatchingConverters(name string) []Converter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := make([]Converter, 0)
	for _, converter := range r.converters {
		if converter.Match(name) {
			matched = append(matched, converter)
		}
	}

	return matched
}

// GetOrCreateGauge gets or creates a gauge metric
func (r *Registry) GetOrCreateGauge(name, help string) prometheus.Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()

	if gauge, exists := r.gauges[name]; exists {
		return gauge
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
	})

	prometheus.MustRegister(gauge)
	r.gauges[name] = gauge
	return gauge
}

func (r *Registry) GetOrCreateInfo(name, help string) prometheus.Gauge {
	gauge := r.GetOrCreateGauge(name, help)
	gauge.Set(1)

	return gauge
}

// GetOrCreateCounter gets or creates a counter metric
func (r *Registry) GetOrCreateCounter(name, help string) prometheus.Counter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if counter, exists := r.counters[name]; exists {
		return counter
	}

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
	})

	prometheus.MustRegister(counter)
	r.counters[name] = counter
	return counter
}

func (r *Registry) GetOrCreateCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	r.mu.Lock()
	defer r.mu.Unlock()

	if counterVec, exists := r.counterVecs[name]; exists {
		return counterVec
	}

	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
	}, labels)

	prometheus.MustRegister(counterVec)
	r.counterVecs[name] = counterVec
	return counterVec
}

// GetOrCreateHistogram gets or creates a histogram metric
func (r *Registry) GetOrCreateHistogram(name, help string, buckets []float64) prometheus.Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()

	if histogram, exists := r.histograms[name]; exists {
		return histogram
	}

	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	})

	prometheus.MustRegister(histogram)
	r.histograms[name] = histogram
	return histogram
}
