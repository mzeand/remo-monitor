# remo-monitor

A Go CLI that retrieves the latest temperature and humidity readings from a Nature Remo Lapis via the Cloud API stores them in InfluxDB, and outputs them as JSON after a successful write.

Each invocation collects once and exits. Use an external scheduler such as cron for periodic collection.

## Requirements

- Go 1.27 or later
- Docker with Docker Compose v2
- A Nature Remo Cloud API access token

You can generate an access token from the [Nature Remo home page](https://home.nature.global/).

## Usage

Create a local configuration and replace the placeholder credentials:

```sh
cp .env.example .env
chmod 600 .env
```

Start InfluxDB and build the collector:

```sh
docker compose up -d influxdb
docker compose ps
go build -o remo-monitor ./cmd/remo-monitor
```

Wait until InfluxDB is healthy, then collect one reading:

```sh
./scripts/collect.sh
```

If you have multiple devices, select one by name or ID:

```sh
./scripts/collect.sh -device-name "Living Room"
./scripts/collect.sh -device-id "device-id"
```

The wrapper loads `.env`; when invoking the binary directly, export `NATURE_REMO_TOKEN`,
`INFLUXDB_TOKEN`, `INFLUXDB_ORG`, and `INFLUXDB_BUCKET` first.
`INFLUXDB_URL` defaults to `http://localhost:8086`.

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

## Storage and retrieval

The `remo` measurement uses `device_id` as a tag and `device_name`,
`temperature_celsius`, and `humidity_percent` as fields. Its timestamp is
`measured_at`, with nanosecond precision. Temperature and humidity may have been
reported at different times; the existing reading model uses the newer timestamp.
Collecting the same device and timestamp again updates that point.

InfluxDB 2.7.12 runs on localhost port 8086. Named Docker volumes persist its data
and configuration across container recreation. `docker compose down` preserves
these volumes; `docker compose down -v` deletes them. Initialization settings only
apply to empty volumes. The default bucket retains data indefinitely.

After collecting, retrieve stored readings (change the bucket if configured):

```sh
docker compose exec influxdb influx query 'from(bucket: "remo") |> range(start: -24h) |> filter(fn: (r) => r._measurement == "remo") |> pivot(rowKey: ["_time", "device_id"], columnKey: ["_field"], valueColumn: "_value")'
```

The container CLI uses the credentials created during initialization. The UI is
also available at http://localhost:8086 using the configured username/password.
For API details, see the [InfluxDB write API documentation](https://docs.influxdata.com/influxdb/v2/write-data/developer-tools/api/).

## Periodic collection with cron

On the host running the built binary, add this using `crontab -e`, replacing the
absolute repository path and selecting a device if necessary:

```cron
*/5 * * * * /absolute/path/remo-monitor/scripts/collect.sh -device-name "Living Room" >> /absolute/path/remo-monitor/collector.log 2>&1
```

The wrapper resolves the binary and `.env` relative to its own location. The host
and InfluxDB must be running. Each API operation has a 10-second timeout; failures
exit nonzero and appear in the log. Configure host log rotation for long-term use.

## Continuous integration

The GitHub Actions workflow in `.github/workflows/ci.yml` runs on pull requests,
pushes to `main`, and manual dispatch. It uses the Go version from `go.mod` to
check Go formatting, shell syntax, unit tests, static analysis, and the CLI build.
The workflow requires no application secrets and disables the live InfluxDB
integration test. It does not run the collector or start database services.

## Integration test

Against a running InfluxDB, export the connection variables above and run:

```sh
INFLUXDB_INTEGRATION=1 go test ./internal/influxdb -run TestIntegrationWriteAndQuery -v
```

This writes a synthetic reading and queries it back, checking device ID, timestamp,
temperature, and humidity. Use a dedicated test bucket; test points are retained.
Without `INFLUXDB_INTEGRATION=1`, normal tests require no server or credentials.

## Codex security settings

The repository's `.codex/config.toml` sets local Codex defaults: workspace-write
sandboxing, disabled sandbox network access, user-reviewed escalation requests,
disabled login shells, and secret environment-variable filtering. Temporary
directories remain writable for development tools.

Start a new Codex session in this trusted project to load the configuration.
Check the active permissions in the client; existing sessions are not proof that
the new defaults are active. CLI overrides and managed policies can change the
effective settings. These are project defaults, not enforced organization policy.

Environment filtering does not prevent reading `.env` or explicitly loading it.
Use a checkout without production secrets for stronger separation. Browser and
connector access have separate controls. Network-dependent Go toolchain downloads
may require approval; run live collection and integration tests deliberately with
test credentials. If Go is configured only in a login startup file, make it
available on the parent process's `PATH`.

See the official [configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)
and [configuration precedence](https://learn.chatgpt.com/docs/config-file/config-basic).
