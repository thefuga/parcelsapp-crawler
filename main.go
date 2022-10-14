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

func (t ParcelsAppTracker) TrackParcels(trackingCodes map[string]string) (chan string, error) {
	results := make(chan string, len(trackingCodes))

	go func() {
		var wg sync.WaitGroup
		for label, code := range trackingCodes {
			wg.Add(1)
			go func(l, c string) {
				defer wg.Done()
				res, _ := t.TrackParcel(l, c)
				results <- fmt.Sprintf(
					"{\"label\":\"%s\",\"code\":\"%s\",\"results\":%s}",
					l, c, string(res),
				)
			}(label, code)
		}

		wg.Wait()
		close(results)
	}()

	return results, nil
}

func (t ParcelsAppTracker) TrackParcel(label, trackingCode string) ([]byte, error) {
	res, err := exec.
		Command(t.cfg.NodePath, t.cfg.CrawlerScript, trackingCode).
		Output()

	if err != nil {
		return nil, err
	}

	// fmt.Printf("{\"label\":\"%s\",\"code\":\"%s\",\"result\":%s}", label, trackingCode, res)
	return res, nil
}

func main() {
	cfg, err := NewConfig("config.json")
	if err != nil {
		panic(err)
	}

	parcelsAppTracker := NewParcelsAppTracker(cfg.Trackers.ParcelsApp)
	// ctx, cancel := context.WithCancel(context.Background())
	// <-every(
	// 	ctx,
	// 	cancel,
	// 	cfg.UpdateEvery*time.Second,
	// func() { parcelsAppTracker.TrackParcels(cfg.TrackingCodes) },
	// )

	resultsChan, _ := parcelsAppTracker.TrackParcels(cfg.TrackingCodes)
	fmt.Print("[")
	count := len(cfg.TrackingCodes)
	i := 0
	for res := range resultsChan {
		fmt.Print(res)
		if i < count-1 {
			fmt.Print(",")
		}
		i++
	}
	fmt.Print("]")
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
			case <-time.After(timeout):
				f()
				continue
			}
		}
	}()

	return done
}
