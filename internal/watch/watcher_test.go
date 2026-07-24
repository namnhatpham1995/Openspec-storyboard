package watch

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

func TestDebounceCoalescesBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		input := make(chan struct{}, 10)
		output := Debounce(ctx, input, 20*time.Millisecond)

		for range 10 {
			input <- struct{}{}
		}
		synctest.Wait()
		<-output

		synctest.Wait()
		select {
		case <-output:
			t.Fatal("burst produced more than one signal")
		default:
		}

		input <- struct{}{}
		<-output
	})
}
