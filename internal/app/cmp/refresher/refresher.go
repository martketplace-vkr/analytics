package refresher

import (
	"context"
	"time"
)

const cmpName = "analytics refresher"

type service interface {
	RefreshAggregates(ctx context.Context) error
}

type Refresher struct {
	interval time.Duration
	service  service
	cancel   context.CancelFunc
}

func New(interval time.Duration, service service) *Refresher {
	if interval <= 0 {
		interval = 15 * time.Minute
	}

	return &Refresher{
		interval: interval,
		service:  service,
	}
}

func (r *Refresher) Start(ctx context.Context) error {
	refreshCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	if err := r.service.RefreshAggregates(refreshCtx); err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			select {
			case <-refreshCtx.Done():
				return
			case <-ticker.C:
				_ = r.service.RefreshAggregates(refreshCtx)
			}
		}
	}()

	return nil
}

func (r *Refresher) Stop(_ context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}

	return nil
}

func (r *Refresher) GetStartTimeout() time.Duration {
	return 30 * time.Second
}

func (r *Refresher) GetStopTimeout() time.Duration {
	return 5 * time.Second
}

func (r *Refresher) GetShutdownDelay() time.Duration {
	return 0
}

func (r *Refresher) GetName() string {
	return cmpName
}
