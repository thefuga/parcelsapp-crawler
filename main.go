package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	TrackingCodes []string `json:"tracking_codes"`
	Trackers      struct {
		ParcelsApp ParcelsAppConfig `json:"parcels_app"`
	} `json:"trackers"`
	UpdateEvery time.Duration `json:"update_every"`
}

func NewConfig(path string) (Config, error) {
	var cfg Config

	file, openErr := os.Open(path)
	if openErr != nil {
		return cfg, openErr
	}

	content, readErr := io.ReadAll(file)
	if readErr != nil {
		return cfg, readErr
	}

	return cfg, json.Unmarshal(content, &cfg)
}

type (
	ParcelsAppTracker struct {
		cfg ParcelsAppConfig
	}

	ParcelsAppConfig struct {
		NodePath      string `json:"node_path"`
		CrawlerScript string `json:"crawler_script"`
	}
)

func NewParcelsAppTracker(cfg ParcelsAppConfig) ParcelsAppTracker {
	return ParcelsAppTracker{cfg: cfg}
}

func (t ParcelsAppTracker) TrackParcels(trackingCodes ...string) error {
	for _, trackingCode := range trackingCodes {
		if err := t.TrackParcel(trackingCode); err != nil {
			return err
		}
	}

	return nil
}

func (t ParcelsAppTracker) TrackParcel(trackingCode string) error {
	result, err := exec.
		Command(t.cfg.NodePath, t.cfg.CrawlerScript, trackingCode).
		Output()

	if err != nil {
		return err
	}

	fmt.Printf("%s\n", result)
	return nil
}

func main() {
	cfg, err := NewConfig("config.json")
	if err != nil {
		panic(err)
	}

	parcelsAppTracker := NewParcelsAppTracker(cfg.Trackers.ParcelsApp)
	ctx, cancel := context.WithCancel(context.Background())

	<-every(
		ctx,
		cancel,
		cfg.UpdateEvery*time.Second,
		func() { parcelsAppTracker.TrackParcels(cfg.TrackingCodes...) },
	)
}

func every(
	ctx context.Context,
	cancel context.CancelFunc,
	timeout time.Duration,
	f func(),
) chan struct{} {
	done := make(chan struct{})
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer func() { done <- struct{}{} }()

		f()

		for {
			select {
			case <-sigs:
				cancel()
				return
			// case <-ctx.Done(): doesn't make much sense
			// 	return
			case <-time.After(timeout):
				f()
				continue
			}
		}
	}()

	return done
}
