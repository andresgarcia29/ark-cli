package cmd

import (
	"context"
	"time"

	"github.com/andresgarcia29/ark-cli/lib/animation"
	services_aws "github.com/andresgarcia29/ark-cli/services/aws"
)

// reportScanProgress keeps the spinner honest during a long scan: it shows how
// many accounts are done and, crucially, says so when AWS starts rate limiting.
// Without it a throttled run is indistinguishable from a hang, because the
// retryer backs off silently.
func reportScanProgress(ctx context.Context, note animation.Progress) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				done, total := services_aws.ScanProgress()
				throttled := services_aws.ThrottleCount()

				switch {
				case throttled > 0 && total > 0:
					note("%d/%d accounts · AWS is rate limiting us (%d throttled), backing off", done, total, throttled)
				case throttled > 0:
					note("AWS is rate limiting us (%d requests throttled), backing off", throttled)
				case total > 0:
					note("%d/%d accounts scanned", done, total)
				}
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
