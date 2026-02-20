package main

import (
	"context"
	"yeelight/pkg/yeelight"
)

func main() {
	ctx := context.Background()
	devices, err := yeelight.Discover(ctx, nil)
	if err != nil {
		panic(err)
	}
	for _, device := range devices {
		device.FetchProps(ctx)

	}

	<-ctx.Done()

}
