package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mzeand/remo-monitor/internal/remo"
)

const requestTimeout = 10 * time.Second

func main() {
	os.Exit(run())
}

func run() int {
	var deviceID string
	var deviceName string
	flag.StringVar(&deviceID, "device-id", "", "Nature Remo device ID to use")
	flag.StringVar(&deviceName, "device-name", "", "Nature Remo device name to use")
	flag.Parse()

	if deviceID != "" && deviceName != "" {
		fmt.Fprintln(os.Stderr, "error: -device-id and -device-name cannot be used together")
		return 2
	}

	token := os.Getenv("NATURE_REMO_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: NATURE_REMO_TOKEN is not set")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	client := remo.NewClient(http.DefaultClient, token)
	reading, err := client.GetReading(ctx, remo.Selector{ID: deviceID, Name: deviceName})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Fprintln(os.Stderr, "error: Nature Remo API request timed out")
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		return 1
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(reading); err != nil {
		fmt.Fprintf(os.Stderr, "error: encode output: %v\n", err)
		return 1
	}
	return 0
}
