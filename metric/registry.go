package metric

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Registry holds all metrics in maps
type Registry struct {
	namespace string

	gauges      map[string]prometheus.Gauge
	gaugeVecs   map[string]*prometheus.GaugeVec
	counters    map[string]prometheus.Counter
	counterVecs map[string]*prometheus.CounterVec
	histograms  map[string]prometheus.Histogram
	mu          sync.RWMutex
}

// NewRegistry creates a new metric registry
func NewRegistry(namespace string) *Registry {
	return &Registry{
		namespace:   namespace,
		gauges:      make(map[string]prometheus.Gauge),
		gaugeVecs:   make(map[string]*prometheus.GaugeVec),
		counters:    make(map[string]prometheus.Counter),
		counterVecs: make(map[string]*prometheus.CounterVec),
		histograms:  make(map[string]prometheus.Histogram),
	}
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
