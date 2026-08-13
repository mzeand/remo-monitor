package remo

import "time"

// Reading is the latest temperature and humidity reported by a Nature Remo.
type Reading struct {
	DeviceID           string    `json:"device_id"`
	DeviceName         string    `json:"device_name"`
	TemperatureCelsius float64   `json:"temperature_celsius"`
	HumidityPercent    float64   `json:"humidity_percent"`
	MeasuredAt         time.Time `json:"measured_at"`
}

// Selector identifies a device. An empty selector is valid when exactly one
// device is registered with the account.
type Selector struct {
	ID   string
	Name string
}

type device struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	NewestEvents map[string]event `json:"newest_events"`
}

type event struct {
	Value     float64   `json:"val"`
	CreatedAt time.Time `json:"created_at"`
}
