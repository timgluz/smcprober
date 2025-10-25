package smartcitizen

import "time"

const (
	DeviceStateOnline   = 1.0
	DeviceStateOffline  = 0.0
	DeviceStateSleeping = 0.5
	DeviceStateUnknown  = -1.0
)

type UserDevice struct {
	ID          int    `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
	KitID       int    `json:"kit_id"`
	MACAddress  string `json:"mac_address"`

	AddedAt       string `json:"added_at"`
	UpdatedAt     string `json:"updated_at"`
	LastReadingAt string `json:"last_reading_at"`
}

type DeviceDetail struct {
	ID          int      `json:"id"`
	UUID        string   `json:"uuid"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	State       string   `json:"state"`
	SystemTags  []string `json:"system_tags"`
	UserTags    []string `json:"user_tags"`

	Owner User       `json:"owner"`
	Data  DeviceData `json:"data"`

	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	LastReadingAt string `json:"last_reading_at"`
}

func (d *DeviceDetail) GetSensorByName(name string) (*DeviceSensor, bool) {
	if d.Data.Sensors == nil {
		return nil, false
	}

	for _, sensor := range d.Data.Sensors {
		if sensor.Name == name {
			return &sensor, true
		}
	}

	return nil, false
}

func (d *DeviceDetail) StateValue() float64 {
	switch d.State {
	case "online", "has_published":
		return DeviceStateOnline
	case "offline":
		return DeviceStateOffline
	case "sleeping":
		return DeviceStateSleeping
	default:
		return DeviceStateUnknown
	}
}

type DeviceData struct {
	Firmware string         `json:"firmware"`
	Location DeviceLocation `json:"location"`

	Sensors []DeviceSensor `json:"sensors"`

	RecordedAt string `json:"recorded_at"`
	AddedAt    string `json:"added_at"`
}

type DeviceLocation struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	IP          string  `json:"ip"`
	Exposure    string  `json:"exposure"`
	Elevation   float64 `json:"elevation"`
	GeoHash     string  `json:"geohash"`
	City        string  `json:"city"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
}

type DeviceSensor struct {
	ID   int    `json:"id"`
	UUID string `json:"uuid"`

	Name        string  `json:"name"`
	Description string  `json:"description"`
	Unit        string  `json:"unit"`
	Value       float64 `json:"value"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (s *DeviceSensor) ToUnix() int64 {
	return ParseTimeToUnix(s.UpdatedAt)
}

func ParseTimeToUnix(timestr string) int64 {
	t, err := time.Parse(time.RFC3339, timestr)
	if err != nil {
		return 0
	}

	return t.Unix()
}
