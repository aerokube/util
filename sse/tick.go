package sse

import (
	"context"
	"os"
	"time"
)

func Tick(broker Broker, notify func(context.Context, Broker), period time.Duration, stop chan os.Signal) {
	ticker := time.NewTicker(period)
	for {
		ctx, cancel := context.WithCancel(context.Background())
		select {
		case <-ticker.C:
			{
				if broker.HasClients() {
					notify(ctx, broker)
				}
			}
		case <-stop:
			{
				cancel()
				ticker.Stop()
			}
		}
	}
}
