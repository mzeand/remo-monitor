package influxdb

import (
	"context"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mzeand/remo-monitor/internal/remo"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWriteReading(t *testing.T) {
	for _, status := range []int{204, 401, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client, err := NewClient(&http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
				if r.Method != "POST" || r.URL.Path != "/api/v2/write" || r.URL.Query().Get("org") != "my org" || r.URL.Query().Get("bucket") != "a&b" || r.URL.Query().Get("precision") != "ns" {
					t.Errorf("unexpected request: %v", r)
				}
				if r.Header.Get("Authorization") != "Token secret" {
					t.Error("missing authorization")
				}
				body, _ := io.ReadAll(r.Body)
				want := "remo,device_id=a\\ b\\,c\\=d device_name=\"Room \\\"A\\\"\",temperature_celsius=24.3,humidity_percent=51 1786592096000000000\n"
				if string(body) != want {
					t.Errorf("body = %q, want %q", body, want)
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
			})}, "http://localhost:8086", "secret", "my org", "a&b")
			if err != nil {
				t.Fatal(err)
			}
			err = client.WriteReading(context.Background(), remo.Reading{DeviceID: "a b,c=d", DeviceName: `Room "A"`, TemperatureCelsius: 24.3, HumidityPercent: 51, MeasuredAt: time.Date(2026, 8, 13, 3, 34, 56, 0, time.UTC)})
			if (err == nil) != (status == 204) {
				t.Fatalf("status %d: error %v", status, err)
			}
		})
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, address := range []string{"", "ftp://host", "http://", "http://user:pass@host", "http://host?x=1"} {
		if _, err := NewClient(nil, address, "token", "org", "bucket"); err == nil {
			t.Errorf("accepted %q", address)
		}
	}
	for _, config := range [][3]string{{"", "org", "bucket"}, {"token", "", "bucket"}, {"token", "org", ""}} {
		if _, err := NewClient(nil, "http://localhost:8086", config[0], config[1], config[2]); err == nil {
			t.Errorf("accepted missing configuration")
		}
	}
}

func TestInvalidReading(t *testing.T) {
	client, _ := NewClient(&http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid reading sent to server")
		return nil, nil
	})}, "http://localhost:8086", "token", "org", "bucket")
	for _, r := range []remo.Reading{
		{DeviceID: "id"},
		{DeviceID: "id\nmalicious", MeasuredAt: time.Now()},
		{DeviceID: "id", MeasuredAt: time.Now(), TemperatureCelsius: math.NaN()},
		{DeviceID: "id", MeasuredAt: time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)},
	} {
		if err := client.WriteReading(context.Background(), r); err == nil {
			t.Error("accepted invalid reading")
		}
	}
}
