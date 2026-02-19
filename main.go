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
		device.SendCommand(ctx, yeelight.Command{
			Method: yeelight.GetProp,
			Params: []any{"*"},
		})
	}

}
