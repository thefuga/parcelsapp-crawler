package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

//go:embed index.js
var trackScript string

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)

	<-every(ctx, cancel, 1*time.Second, func() { trackParcelInNode("LX897146572CN") })
}

func trackParcelInNode(tracker string) {
	result, err := exec.
		Command("add path here", "index.js", tracker).
		Output()

	if err != nil {
		panic("command error: " + err.Error())
	}

	fmt.Printf("%s\n", result)
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
			case <-ctx.Done():
				return
			case <-time.After(timeout):
				f()
				continue
			}
		}
	}()

	return done
}

// func trackParcel(code string) {
// 	iso := v8.NewIsolate()
// 	ctx := v8.NewContext(iso)
// 	value, err := ctx.RunScript(fmt.Sprintf(trackScript), "track.js")

// 	if err != nil {
// 		panic("error: " + err.Error())
// 	}

// 	promise, err := value.AsPromise()
// 	if err != nil {
// 		panic("error: " + err.Error())
// 	}

// 	promise.
// 		Then(func(info *v8.FunctionCallbackInfo) *v8.Value {
// 			fmt.Println(info.Args())
// 			return nil
// 		}).
// 		Catch(func(info *v8.FunctionCallbackInfo) *v8.Value {
// 			fmt.Println(info.Args())
// 			return nil
// 		})
// }
