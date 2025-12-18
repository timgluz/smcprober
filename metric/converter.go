package metric

import (
	"reflect"
	"sync"
)

type Converter interface {
	Match(name string) bool
	Convert(Registry, any) error
}

type CombinedConverter struct {
	mu sync.RWMutex

	converters []Converter
}

func NewCombinedConverter(converters ...Converter) *CombinedConverter {
	return &CombinedConverter{converters: converters}
}

func (c *CombinedConverter) Add(converters ...Converter) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.converters = append(c.converters, converters...)
}

func (c *CombinedConverter) Match(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, converter := range c.converters {
		if converter.Match(name) {
			return true
		}
	}
	return false
}

func (c *CombinedConverter) Convert(registry Registry, data any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, converter := range c.converters {
		if !converter.Match(getTypeName(data)) {
			continue
		}

		if err := converter.Convert(registry, data); err != nil {
			return err
		}
	}
	return nil
}

func getTypeName(data any) string {
	return reflect.TypeOf(data).Name()
}
