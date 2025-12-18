package metric

import (
	"reflect"
	"sync"
)

// Converter defines the contract for types that can register metrics in a Registry
// from arbitrary data values. Implementations typically inspect the type or name
// of the provided data and, when applicable, populate the given Registry with
// one or more metrics derived from that data.
type Converter interface {
	// Match reports whether this Converter is able to handle a metric or data
	// source identified by the provided name. Callers use Match to determine
	// which converter(s) should be invoked for a particular type or metric name.
	Match(name string) bool

	// Convert registers metrics in the supplied Registry based on the provided
	// data value. It should only be called when Match has returned true for the
	// corresponding type or name. Implementations must return a non-nil error
	// if the conversion or registration fails; otherwise they should return nil.
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
