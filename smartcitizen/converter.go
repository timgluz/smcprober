package smartcitizen

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/timgluz/smcprober/metric"
)

var (
	ErrInvalidDataType = fmt.Errorf("invalid data type for converter")
)

const (
	DeviceDetailType = "DeviceDetail"
	DeviceSensorType = "DeviceSensor"
)

type DeviceInfoConverter struct {
	metricName string
}

func NewDeviceInfoConverter(metricName string) *DeviceInfoConverter {

	return &DeviceInfoConverter{metricName}
}

func (c *DeviceInfoConverter) Match(name string) bool {
	return name == DeviceDetailType
}

func (c *DeviceInfoConverter) Convert(registry metric.Registry, data any) error {
	device, ok := data.(DeviceDetail)
	if !ok {
		return fmt.Errorf("%w: Invalid data type %v", ErrInvalidDataType, reflect.TypeOf(data))
	}

	labels := prometheus.Labels{
		"uuid":        device.UUID,
		"name":        device.Name,
		"description": device.Description,
	}

	gauge := registry.GetOrCreateGaugeVec(
		c.metricName,
		"Static information about Smart Citizen devices",
		[]string{"uuid", "name", "description"},
	)

	gauge.With(labels).Set(1)
	return nil
}

type DeviceStateConverter struct {
	metricName string
}

func NewDeviceStateConverter(metricName string) *DeviceStateConverter {
	return &DeviceStateConverter{metricName}
}

func (c *DeviceStateConverter) Match(name string) bool {
	return name == DeviceDetailType
}

func (c *DeviceStateConverter) Convert(registry metric.Registry, data any) error {
	device, ok := data.(DeviceDetail)
	if !ok {
		return ErrInvalidDataType
	}

	gauge := registry.GetOrCreateGaugeVec(
		c.metricName+"_has_published",
		"Indicates whether the device has published data (1) or not (0)",
		[]string{"uuid", "name"},
	)

	labels := prometheus.Labels{
		"uuid": device.UUID,
		"name": device.Name,
	}

	gauge.With(labels).Set(device.StateValue())
	return nil
}

type DeviceSensorConverter struct {
	metricName    string
	sensorMapping *metric.SensorMetricMapping
}

func NewDeviceSensorConverter(metricName string, sensorMapping *metric.SensorMetricMapping) *DeviceSensorConverter {
	return &DeviceSensorConverter{
		metricName:    metricName,
		sensorMapping: sensorMapping,
	}
}

func (c *DeviceSensorConverter) Match(name string) bool {
	return name == DeviceSensorType
}
func (c *DeviceSensorConverter) Convert(registry metric.Registry, data any) error {
	sensor, ok := data.(DeviceSensor)
	if !ok {
		return ErrInvalidDataType
	}

	// Default to the generic state metric name
	metricName := c.metricName + "_state"
	sensorMetric, exists := c.sensorMapping.Get(sensor.Name)

	// Use the mapped metric name only if the mapping exists and has a non-empty Metric field
	if exists && sensorMetric.Metric != "" {
		metricName = c.metricName + "_" + sensorMetric.MetricName()
	}

	gauge := registry.GetOrCreateGaugeVec(
		metricName,
		"Current sensor value",
		[]string{"id", "uuid", "name", "device"},
	)

	labels := prometheus.Labels{
		"id":     strconv.Itoa(sensor.ID),
		"uuid":   sensor.UUID,
		"name":   sensor.Name,
		"device": sensor.DeviceUUID,
	}

	gauge.With(labels).Set(sensor.Value)
	return nil
}

type DeviceSensorInfoConverter struct {
	metricName string
}

func NewDeviceSensorInfoConverter(metricName string) *DeviceSensorInfoConverter {
	return &DeviceSensorInfoConverter{metricName}
}

func (c *DeviceSensorInfoConverter) Match(name string) bool {
	return name == DeviceSensorType
}

func (c *DeviceSensorInfoConverter) Convert(registry metric.Registry, data any) error {
	sensor, ok := data.(DeviceSensor)
	if !ok {
		return ErrInvalidDataType
	}

	labels := prometheus.Labels{
		"id":          strconv.Itoa(sensor.ID),
		"uuid":        sensor.UUID,
		"name":        sensor.Name,
		"unit":        sensor.Unit,
		"description": sensor.Description,
	}

	gauge := registry.GetOrCreateGaugeVec(
		c.metricName,
		"Static information about Smart Citizen device sensors",
		[]string{"id", "uuid", "name", "unit", "description"},
	)

	gauge.With(labels).Set(1)
	return nil
}
