// Package influxdb stores sensor readings using the InfluxDB v2 write API.
package influxdb

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mzeand/remo-monitor/internal/remo"
)

type Client struct {
	httpClient *http.Client
	endpoint   string
	token      string
}

func NewClient(httpClient *http.Client, address, token, org, bucket string) (*Client, error) {
	if token == "" || org == "" || bucket == "" {
		return nil, fmt.Errorf("INFLUXDB_TOKEN, INFLUXDB_ORG and INFLUXDB_BUCKET must be set")
	}
	u, err := url.Parse(address)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("INFLUXDB_URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v2/write"
	q := url.Values{"org": {org}, "bucket": {bucket}, "precision": {"ns"}}
	u.RawQuery = q.Encode()
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient, endpoint: u.String(), token: token}, nil
}

func (c *Client) WriteReading(ctx context.Context, r remo.Reading) error {
	if r.DeviceID == "" || strings.ContainsAny(r.DeviceID+r.DeviceName, "\r\n") {
		return fmt.Errorf("invalid device identity for InfluxDB")
	}
	if r.MeasuredAt.IsZero() || !r.MeasuredAt.Equal(time.Unix(0, r.MeasuredAt.UnixNano())) {
		return fmt.Errorf("measurement timestamp is missing or outside nanosecond range")
	}
	if math.IsNaN(r.TemperatureCelsius) || math.IsInf(r.TemperatureCelsius, 0) || math.IsNaN(r.HumidityPercent) || math.IsInf(r.HumidityPercent, 0) {
		return fmt.Errorf("sensor values must be finite")
	}
	escape := strings.NewReplacer("\\", "\\\\", " ", "\\ ", ",", "\\,", "=", "\\=")
	quote := strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
	line := fmt.Sprintf("remo,device_id=%s device_name=\"%s\",temperature_celsius=%s,humidity_percent=%s %d\n",
		escape.Replace(r.DeviceID), quote.Replace(r.DeviceName),
		strconv.FormatFloat(r.TemperatureCelsius, 'g', -1, 64), strconv.FormatFloat(r.HumidityPercent, 'g', -1, 64), r.MeasuredAt.UnixNano())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(line))
	if err != nil {
		return fmt.Errorf("create InfluxDB request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("write InfluxDB reading: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("InfluxDB write returned HTTP %d", resp.StatusCode)
	}
	return nil
}
