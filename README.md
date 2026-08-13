# remo-monitor

Nature Remo LapisからCloud API経由で最新の温度・湿度を取得し、JSONで出力するGo CLIです。

現時点では、定期実行およびAWS IoT CoreへのMQTT送信は実装していません。

## 必要なもの

- Go 1.25以降
- Nature Remo Cloud APIのアクセストークン

アクセストークンは[Nature Remoのホーム画面](https://home.nature.global/)から発行します。

## 実行

登録デバイスが1台の場合:

```sh
NATURE_REMO_TOKEN="your-access-token" go run ./cmd/remo-monitor
```

複数台の場合は名前またはIDで選択します。

```sh
NATURE_REMO_TOKEN="your-access-token" go run ./cmd/remo-monitor -device-name "Living Room"
NATURE_REMO_TOKEN="your-access-token" go run ./cmd/remo-monitor -device-id "device-id"
```

出力例:

```json
{
  "device_id": "device-id",
  "device_name": "Living Room",
  "temperature_celsius": 24.3,
  "humidity_percent": 51,
  "measured_at": "2026-08-13T03:34:56Z"
}
```

`measured_at`には、温度と湿度のうち新しい方の測定日時を出力します。ログとエラーは標準エラー出力へ送られます。

## テスト

```sh
go test ./...
go vet ./...
```

## ビルド

ローカル向け:

```sh
go build -o remo-monitor ./cmd/remo-monitor
```

64-bit Raspberry Pi向け:

```sh
GOOS=linux GOARCH=arm64 go build -o remo-monitor-linux-arm64 ./cmd/remo-monitor
```
