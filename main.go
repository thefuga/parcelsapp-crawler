package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
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
		ChromePath string `json:"node_path"`
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
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "chrome"),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	if t.cfg.ChromePath != "" {
		opts = append(opts, chromedp.ExecPath(t.cfg.ChromePath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(
		allocCtx,
	)
	defer cancel()

	var requestID network.RequestID
	done := make(chan struct{})
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch event := ev.(type) {
		case *network.EventRequestWillBeSent:
			if strings.Contains(event.Request.URL, "api/v2/parcels") {
				requestID = event.RequestID
			}
		case *network.EventLoadingFinished:
			if requestID == event.RequestID {
				close(done)
			}
		}
	})

	err := chromedp.Run(
		ctx,
		chromedp.Navigate("https://parcelsapp.com/widget"),
		chromedp.SetValue("#track-input", trackingCode),
		chromedp.Click("#track-button", chromedp.NodeVisible),
	)

	<-done

	var responseBody []byte
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		responseBody, err = network.GetResponseBody(requestID).Do(ctx)
		return err
	})); err != nil {
		log.Fatal(err)
	}

	return responseBody, err
}

func main() {
	cfg, err := NewConfig("config.json")
	if err != nil {
		panic(err)
	}

	parcelsAppTracker := NewParcelsAppTracker(cfg.Trackers.ParcelsApp)

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
