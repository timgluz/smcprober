package metric

import (
	"log/slog"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Registry interface {
	GetCollectorByName(name string) (prometheus.Collector, bool)
	Register(name string, collector prometheus.Collector)

	// Constructors / Getters
	GetOrCreateGauge(name, help string) prometheus.Gauge
	GetOrCreateGaugeVec(name, help string, labels []string) *prometheus.GaugeVec
	GetOrCreateCounter(name, help string) prometheus.Counter
	GetOrCreateCounterVec(name, help string, labels []string) *prometheus.CounterVec
	GetOrCreateHistogram(name, help string, buckets []float64) prometheus.Histogram
	GetOrCreateHistogramVec(name, help string, buckets []float64, labels []string) *prometheus.HistogramVec
}

// NamespacedRegistry holds all metrics in maps
// and provides methods to get or create them from sensor data using converters
type NamespacedRegistry struct {
	namespace string
	mu        sync.RWMutex

	// Track registered collectors to avoid re-registration
	collectors map[string]prometheus.Collector

	logger *slog.Logger
}

// NewNamespacedRegistry creates a new metric registry
func NewNamespacedRegistry(namespace string, logger *slog.Logger) *NamespacedRegistry {
	return &NamespacedRegistry{
		namespace:  namespace,
		collectors: make(map[string]prometheus.Collector),
		logger:     logger,
	}
}

func (r *NamespacedRegistry) GetCollectorByName(name string) (prometheus.Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	collector, exists := r.collectors[name]
	return collector, exists
}

func (r *NamespacedRegistry) Register(name string, collector prometheus.Collector) {
	if _, exists := r.GetCollectorByName(name); exists {
		return
	}

	if err := prometheus.Register(collector); err != nil {
		r.logger.Error("Failed to register collector", "name", name, "error", err)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[name] = collector
}

// GetOrCreateGauge gets or creates a gauge metric
func (r *NamespacedRegistry) GetOrCreateGauge(name, help string) prometheus.Gauge {
	if gauge, exists := r.GetCollectorByName(name); exists {
		return gauge.(prometheus.Gauge)
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
	})

	r.Register(name, gauge)
	return gauge
}

// GetOrCreateGaugeVec gets or creates a gauge vector metric
func (r *NamespacedRegistry) GetOrCreateGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {

	if gaugeVec, exists := r.GetCollectorByName(name); exists {
		return gaugeVec.(*prometheus.GaugeVec)
	}

	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
	}, labels)

	r.Register(name, gaugeVec)
	return gaugeVec
}

// GetOrCreateCounter gets or creates a counter metric
func (r *NamespacedRegistry) GetOrCreateCounter(name, help string) prometheus.Counter {
	if counter, exists := r.GetCollectorByName(name); exists {
		return counter.(prometheus.Counter)
	}

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
	})

	r.Register(name, counter)
	return counter
}

func (r *NamespacedRegistry) GetOrCreateCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	if counterVec, exists := r.GetCollectorByName(name); exists {
		return counterVec.(*prometheus.CounterVec)
	}

	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
	}, labels)

	r.Register(name, counterVec)
	return counterVec
}

// GetOrCreateHistogram gets or creates a histogram metric
func (r *NamespacedRegistry) GetOrCreateHistogram(name, help string, buckets []float64) prometheus.Histogram {
	if histogram, exists := r.GetCollectorByName(name); exists {
		return histogram.(prometheus.Histogram)
	}

	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	})

	r.Register(name, histogram)
	return histogram
}

// GetOrCreateHistogramVec gets or creates a histogram vector metric
func (r *NamespacedRegistry) GetOrCreateHistogramVec(name, help string, buckets []float64, labels []string) *prometheus.HistogramVec {
	if histogramVec, exists := r.GetCollectorByName(name); exists {
		return histogramVec.(*prometheus.HistogramVec)
	}

	histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: r.namespace,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}, labels)

	r.Register(name, histogramVec)
	return histogramVec
}
