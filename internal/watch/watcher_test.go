package watch

import (
	"context"
	"testing"
	"time"
)

func TestDebounceCoalescesBurst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan struct{}, 10)
	output := Debounce(ctx, input, 20*time.Millisecond)

	for range 10 {
		input <- struct{}{}
	}
	select {
	case <-output:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for debounced signal")
	}
	select {
	case <-output:
		t.Fatal("burst produced more than one signal")
	case <-time.After(40 * time.Millisecond):
	}

	input <- struct{}{}
	select {
	case <-output:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("second burst did not produce a signal")
	}
}
