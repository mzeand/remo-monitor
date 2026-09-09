package influxdb

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mzeand/remo-monitor/internal/remo"
)

// Run against a dedicated test bucket; this writes a synthetic reading.
func TestIntegrationWriteAndQuery(t *testing.T) {
	if os.Getenv("INFLUXDB_INTEGRATION") != "1" {
		t.Skip("set INFLUXDB_INTEGRATION=1 to test a running InfluxDB")
	}
	address, token, org, bucket := os.Getenv("INFLUXDB_URL"), os.Getenv("INFLUXDB_TOKEN"), os.Getenv("INFLUXDB_ORG"), os.Getenv("INFLUXDB_BUCKET")
	client, err := NewClient(nil, address, token, org, bucket)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	now := time.Now().UTC()
	id := fmt.Sprintf("integration-%d", now.UnixNano())
	if err := client.WriteReading(ctx, remo.Reading{DeviceID: id, DeviceName: "Integration test", TemperatureCelsius: 24.3, HumidityPercent: 51, MeasuredAt: now}); err != nil {
		t.Fatal(err)
	}
	query := fmt.Sprintf(`from(bucket: %q) |> range(start: -1h) |> filter(fn: (r) => r._measurement == "remo" and r.device_id == %q) |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")`, bucket, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(address, "/")+"/api/v2/query?org="+url.QueryEscape(org), strings.NewReader(query))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/vnd.flux")
	req.Header.Set("Accept", "application/csv")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("query HTTP %d", resp.StatusCode)
	}
	rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected header and one row, got %s", body)
	}
	got := map[string]string{}
	for i, column := range rows[0] {
		got[column] = rows[1][i]
	}
	for field, want := range map[string]string{"device_id": id, "device_name": "Integration test", "_time": now.Format(time.RFC3339Nano), "temperature_celsius": "24.3", "humidity_percent": "51"} {
		if got[field] != want {
			t.Errorf("%s = %q, want %q", field, got[field], want)
		}
	}
}
