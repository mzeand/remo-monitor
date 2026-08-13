package remo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultBaseURL = "https://api.nature.global"
	maxErrorBody   = 1024
)

// Client reads sensor data from the Nature Remo Cloud API.
type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

// NewClient creates a Cloud API client.
func NewClient(httpClient *http.Client, token string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient, token: token, baseURL: defaultBaseURL}
}

// GetReading returns the latest temperature and humidity for the selected device.
func (c *Client) GetReading(ctx context.Context, selector Selector) (Reading, error) {
	devices, err := c.getDevices(ctx)
	if err != nil {
		return Reading{}, err
	}

	selected, err := selectDevice(devices, selector)
	if err != nil {
		return Reading{}, err
	}

	temperature, ok := selected.NewestEvents["te"]
	if !ok {
		return Reading{}, fmt.Errorf("device %q has no temperature reading", selected.Name)
	}
	humidity, ok := selected.NewestEvents["hu"]
	if !ok {
		return Reading{}, fmt.Errorf("device %q has no humidity reading", selected.Name)
	}

	measuredAt := temperature.CreatedAt
	if humidity.CreatedAt.After(measuredAt) {
		measuredAt = humidity.CreatedAt
	}
	return Reading{
		DeviceID:           selected.ID,
		DeviceName:         selected.Name,
		TemperatureCelsius: temperature.Value,
		HumidityPercent:    humidity.Value,
		MeasuredAt:         measuredAt,
	}, nil
}

func (c *Client) getDevices(ctx context.Context) ([]device, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/1/devices", nil)
	if err != nil {
		return nil, fmt.Errorf("create Nature Remo API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Nature Remo API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		detail := strings.TrimSpace(string(body))
		if detail == "" {
			detail = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("Nature Remo API returned HTTP %d: %s", resp.StatusCode, detail)
	}

	var devices []device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, fmt.Errorf("decode Nature Remo API response: %w", err)
	}
	return devices, nil
}

func selectDevice(devices []device, selector Selector) (device, error) {
	if selector.ID != "" || selector.Name != "" {
		var matches []device
		for _, candidate := range devices {
			if selector.ID != "" && candidate.ID == selector.ID ||
				selector.Name != "" && candidate.Name == selector.Name {
				matches = append(matches, candidate)
			}
		}
		if len(matches) == 0 {
			return device{}, fmt.Errorf("selected Nature Remo device was not found")
		}
		if len(matches) > 1 {
			return device{}, fmt.Errorf("multiple devices are named %q; select one with -device-id", selector.Name)
		}
		return matches[0], nil
	}

	switch len(devices) {
	case 0:
		return device{}, fmt.Errorf("no Nature Remo devices are registered")
	case 1:
		return devices[0], nil
	default:
		return device{}, fmt.Errorf("multiple Nature Remo devices are registered; specify -device-name or -device-id")
	}
}
