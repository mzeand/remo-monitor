# remo-monitor

A Go CLI that retrieves the latest temperature and humidity readings from a Nature Remo Lapis via the Cloud API and outputs them as JSON.

Scheduled execution and MQTT publishing to AWS IoT Core are not currently implemented.

## Requirements

- Go 1.25 or later
- A Nature Remo Cloud API access token

You can generate an access token from the [Nature Remo home page](https://home.nature.global/).

## Usage

If you have one registered device:

```sh
NATURE_REMO_TOKEN="your-access-token" go run ./cmd/remo-monitor
```

If you have multiple devices, select one by name or ID:

```sh
NATURE_REMO_TOKEN="your-access-token" go run ./cmd/remo-monitor -device-name "Living Room"
NATURE_REMO_TOKEN="your-access-token" go run ./cmd/remo-monitor -device-id "device-id"
```

Example output:

```json
{
  "device_id": "device-id",
  "device_name": "Living Room",
  "temperature_celsius": 24.3,
  "humidity_percent": 51,
  "measured_at": "2026-08-13T03:34:56Z"
}
```

`measured_at` contains the more recent timestamp of the temperature and humidity readings. Logs and errors are written to standard error.

## Testing

```sh
go test ./...
go vet ./...
```

## Building

For your local platform:

```sh
go build -o remo-monitor ./cmd/remo-monitor
```

For a 64-bit Raspberry Pi:

```sh
GOOS=linux GOARCH=arm64 go build -o remo-monitor-linux-arm64 ./cmd/remo-monitor
```
