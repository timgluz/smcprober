package smartcitizen

import (
	"fmt"
	"strconv"

	"github.com/gosimple/slug"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrInvalidDataType = fmt.Errorf("invalid data type for converter")
)

const (
	DeviceDetailType = "DeviceDetail"
	DeviceSensorType = "DeviceSensor"
)

type DeviceInfoConverter struct {
	gauge *prometheus.GaugeVec
}

func NewDeviceInfoConverter() *DeviceInfoConverter {
	return &DeviceInfoConverter{
		gauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smartcitizen_device_info",
				Help: "Static information about Smart Citizen devices",
			},
			[]string{"uuid", "name", "description"},
		),
	}
}
func (c *DeviceInfoConverter) Name() string {
	return "DeviceInfoConverter"
}

func (c *DeviceInfoConverter) Match(name string) bool {
	return name == DeviceDetailType
}

func (c *DeviceInfoConverter) Convert(data any) (prometheus.Collector, error) {
	device, ok := data.(DeviceDetail)
	if !ok {
		return nil, ErrInvalidDataType
	}

	labels := prometheus.Labels{
		"uuid":        device.UUID,
		"name":        device.Name,
		"description": device.Description,
	}

	c.gauge.With(labels).Set(1)
	return c.gauge, nil
}

type DeviceSensorConverter struct {
	gauge *prometheus.GaugeVec
}

func NewDeviceSensorConverter() *DeviceSensorConverter {
	return &DeviceSensorConverter{
		gauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smartcitizen_sensor_state",
				Help: "Current sensor value",
			},
			[]string{"id", "uuid", "name"},
		),
	}
}

func (c *DeviceSensorConverter) Name() string {
	return "DeviceSensorConverter"
}

func (c *DeviceSensorConverter) Match(name string) bool {
	return name == DeviceSensorType
}
func (c *DeviceSensorConverter) Convert(data any) (prometheus.Collector, error) {
	sensor, ok := data.(DeviceSensor)
	if !ok {
		return nil, ErrInvalidDataType
	}

	labels := prometheus.Labels{
		"id":   strconv.Itoa(sensor.ID),
		"uuid": sensor.UUID,
		"name": slug.Make(sensor.Name),
	}

	c.gauge.With(labels).Set(sensor.Value)
	return c.gauge, nil
}

type DeviceSensorInfoConverter struct {
	gauge *prometheus.GaugeVec
}

func NewDeviceSensorInfoConverter() *DeviceSensorInfoConverter {
	return &DeviceSensorInfoConverter{
		gauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smartcitizen_sensor_info",
				Help: "Static information about Smart Citizen device sensors",
			},
			[]string{"id", "uuid", "name", "unit", "description"},
		),
	}
}

func (c *DeviceSensorInfoConverter) Name() string {
	return "DeviceSensorInfoConverter"
}

func (c *DeviceSensorInfoConverter) Match(name string) bool {
	return name == DeviceSensorType
}

func (c *DeviceSensorInfoConverter) Convert(data any) (prometheus.Collector, error) {
	sensor, ok := data.(DeviceSensor)
	if !ok {
		return nil, ErrInvalidDataType
	}

	labels := prometheus.Labels{
		"id":          strconv.Itoa(sensor.ID),
		"uuid":        sensor.UUID,
		"name":        sensor.Name,
		"unit":        sensor.Unit,
		"description": sensor.Description,
	}

	c.gauge.With(labels).Set(1)
	return c.gauge, nil
}
