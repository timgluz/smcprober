package metric

import (
	"fmt"
	"sync"
)

type MetricMappingItem struct {
	Metric   string `json:"metric"`
	Category string `json:"category"`
}

func (m MetricMappingItem) MetricName() string {
	return fmt.Sprintf("%s_%s", m.Category, m.Metric)
}

type SensorMetricMapping struct {
	mu sync.RWMutex

	items map[string]MetricMappingItem
}

func NewSensorMetricMapping() *SensorMetricMapping {
	return &SensorMetricMapping{
		items: make(map[string]MetricMappingItem),
	}
}

func (m *SensorMetricMapping) Add(sensorName string, item MetricMappingItem) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[sensorName] = item
}

func (m *SensorMetricMapping) Get(sensorName string) (MetricMappingItem, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.items[sensorName]
	return item, exists
}
