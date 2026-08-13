package remo

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func testHTTPClient(statusCode int, body string, check func(*http.Request)) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if check != nil {
			check(r)
		}
		return &http.Response{
			StatusCode: statusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})}
}

func TestGetReading(t *testing.T) {
	t.Parallel()

	body := `[
		{"id":"other","name":"Bedroom","newest_events":{}},
		{"id":"lapis-1","name":"Living Room","newest_events":{
			"te":{"val":24.3,"created_at":"2026-08-13T03:34:55Z"},
			"hu":{"val":51,"created_at":"2026-08-13T03:34:56Z"}
		}}
	]`
	httpClient := testHTTPClient(http.StatusOK, body, func(r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/1/devices" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
	})

	client := NewClient(httpClient, "test-token")
	got, err := client.GetReading(context.Background(), Selector{Name: "Living Room"})
	if err != nil {
		t.Fatal(err)
	}
	if got.DeviceID != "lapis-1" || got.DeviceName != "Living Room" {
		t.Errorf("device = %q, %q", got.DeviceID, got.DeviceName)
	}
	if got.TemperatureCelsius != 24.3 || got.HumidityPercent != 51 {
		t.Errorf("values = %v, %v", got.TemperatureCelsius, got.HumidityPercent)
	}
	wantTime := time.Date(2026, 8, 13, 3, 34, 56, 0, time.UTC)
	if !got.MeasuredAt.Equal(wantTime) {
		t.Errorf("MeasuredAt = %v, want %v", got.MeasuredAt, wantTime)
	}
}

func TestGetReadingErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		response   string
		statusCode int
		selector   Selector
		wantError  string
	}{
		{name: "API error", statusCode: http.StatusTooManyRequests, response: `{"message":"rate limited"}`, wantError: "HTTP 429"},
		{name: "invalid JSON", statusCode: http.StatusOK, response: `{`, wantError: "decode"},
		{name: "no devices", statusCode: http.StatusOK, response: `[]`, wantError: "no Nature Remo devices"},
		{name: "ambiguous devices", statusCode: http.StatusOK, response: `[{"id":"1"},{"id":"2"}]`, wantError: "multiple Nature Remo devices"},
		{name: "not found", statusCode: http.StatusOK, response: `[{"id":"1","name":"Room"}]`, selector: Selector{ID: "missing"}, wantError: "not found"},
		{name: "missing temperature", statusCode: http.StatusOK, response: `[{"id":"1","name":"Room","newest_events":{"hu":{"val":50}}}]`, wantError: "no temperature"},
		{name: "missing humidity", statusCode: http.StatusOK, response: `[{"id":"1","name":"Room","newest_events":{"te":{"val":20}}}]`, wantError: "no humidity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewClient(testHTTPClient(tt.statusCode, tt.response, nil), "token")
			_, err := client.GetReading(context.Background(), tt.selector)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}
