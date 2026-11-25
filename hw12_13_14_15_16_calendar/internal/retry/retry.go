package retry

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultInitial = time.Second
	defaultMax     = 30 * time.Second
)

type Logger interface {
	Error(msg string)
}

type Options struct {
	Initial time.Duration
	Max     time.Duration
}

func Do(ctx context.Context, logger Logger, options Options, operation func() error) error {
	initial := options.Initial
	if initial <= 0 {
		initial = defaultInitial
	}

	maxBackoff := options.Max
	if maxBackoff < initial {
		maxBackoff = initial
	}

	backoff := initial
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := operation()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		logger.Error(fmt.Sprintf("operation failed: %v; retrying in %s", err, backoff))

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			return ctx.Err()
		case <-timer.C:
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
