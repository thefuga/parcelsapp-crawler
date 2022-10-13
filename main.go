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
	"sync"
	"syscall"
	"time"
)

type Config struct {
	TrackingCodes map[string]string `json:"tracking_codes"`
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

func (t ParcelsAppTracker) TrackParcels(trackingCodes map[string]string) error {
	var wg sync.WaitGroup

	wg.Add(len(trackingCodes))
	for label, code := range trackingCodes {
		go func(l, c string) {
			defer wg.Done()
			_ = t.TrackParcel(l, c)
		}(label, code)

	}

	wg.Wait()
	return nil
}

func (t ParcelsAppTracker) TrackParcel(label, trackingCode string) error {
	res, err := exec.
		Command(t.cfg.NodePath, t.cfg.CrawlerScript, trackingCode).
		Output()

	if err != nil {
		return err
	}

	fmt.Printf("{\"label\":\"%s\",\"code\":\"%s\",\"result\":%s}", label, trackingCode, res)
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
		func() { parcelsAppTracker.TrackParcels(cfg.TrackingCodes) },
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

		if timeout == 0 {
			return
		}

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
