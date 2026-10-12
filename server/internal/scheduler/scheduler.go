package scheduler

import (
	"context"
	"time"

	"bp-server/internal/model"
	"bp-server/internal/service"
)

// Run ticks once a minute (single instance preferred; sent_log dedup protects duplicates).
func Run(ctx context.Context, app *service.App) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		now := time.Now().In(model.CST)
		app.Tick(ctx, now)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
