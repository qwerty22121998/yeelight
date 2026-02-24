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
		_, err := device.SendCommand(ctx, yeelight.C(yeelight.BgStartCf, yeelight.ColorFlow{
			Count:  20,
			Action: yeelight.FlowOff,
			Expression: []yeelight.FlowExpression{
				{
					Duration:   1000,
					Mode:       yeelight.FlowRGB,
					Value:      yeelight.RGBToInt(255, 0, 0),
					Brightness: 100,
				},
				{
					Duration:   1000,
					Mode:       yeelight.FlowRGB,
					Value:      yeelight.RGBToInt(0, 255, 0),
					Brightness: 100,
				},
				{
					Duration:   1000,
					Mode:       yeelight.FlowRGB,
					Value:      yeelight.RGBToInt(0, 0, 255),
					Brightness: 100,
				},
			},
		}.Build()...))
		if err != nil {
			panic(err)
		}
	}

	<-ctx.Done()

}
