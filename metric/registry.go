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

	// Track registered collectors to avoid re-registration
	registeredCollectors map[prometheus.Collector]bool

	logger *slog.Logger
}

// NewRegistry creates a new metric registry
func NewRegistry(namespace string, logger *slog.Logger) *Registry {
	return &Registry{
		namespace:            namespace,
		converters:           make([]Converter, 0),
		gauges:               make(map[string]prometheus.Gauge),
		gaugeVecs:            make(map[string]*prometheus.GaugeVec),
		counters:             make(map[string]prometheus.Counter),
		counterVecs:          make(map[string]*prometheus.CounterVec),
		histograms:           make(map[string]prometheus.Histogram),
		registeredCollectors: make(map[prometheus.Collector]bool),
		logger:               logger,
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
		collector, err := converter.Convert(data)
		if err != nil {
			return err
		}

		// Check if this collector has already been registered
		r.mu.Lock()
		alreadyRegistered := r.registeredCollectors[collector]

		if !alreadyRegistered {
			// First time seeing this collector, register it
			if err := prometheus.Register(collector); err != nil {
				r.mu.Unlock()
				r.logger.Error("Failed to register metric",
					"converter", converter.Name(), "error", err,
				)
				return err
			}
			// Mark as registered
			r.registeredCollectors[collector] = true
			r.logger.Debug("Registered new collector", "converter", converter.Name())
		}
		r.mu.Unlock()

		// Note: The converter's Convert() method already updated the metric values
		// via gauge.With(labels).Set(value), so we don't need to do anything else
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
