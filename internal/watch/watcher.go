package watch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher recursively watches one project's openspec directory and emits a
// single dirty callback after each quiet debounce window.
type Watcher struct {
	root    string
	delay   time.Duration
	watcher *fsnotify.Watcher
}

// New creates a recursive project watcher. The caller must run Run.
func New(projectRoot string, delay time.Duration) (*Watcher, error) {
	root, err := filepath.Abs(filepath.Join(projectRoot, "openspec"))
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("watching openspec directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("watching openspec directory: %s is not a directory", root)
	}
	backend, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	result := &Watcher{root: root, delay: delay, watcher: backend}
	if err := result.addTree(root); err != nil {
		_ = backend.Close()
		return nil, err
	}
	return result, nil
}

// Run blocks until ctx is cancelled or the filesystem backend fails.
func (w *Watcher) Run(ctx context.Context, dirty func()) error {
	defer w.watcher.Close()
	raw := make(chan struct{}, 1)
	debounced := Debounce(ctx, raw, w.delay)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-debounced:
			dirty()
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = w.addTree(event.Name)
				}
			}
			select {
			case raw <- struct{}{}:
			default:
			}
		case err, ok := <-w.watcher.Errors:
			if ok && err != nil && !errors.Is(err, os.ErrClosed) {
				return fmt.Errorf("filesystem watcher: %w", err)
			}
		}
	}
}

func (w *Watcher) addTree(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				return fmt.Errorf("watching %s: %w", path, err)
			}
		}
		return nil
	})
}

// Debounce coalesces any burst of input signals into one output signal after
// delay has elapsed without another input.
func Debounce(ctx context.Context, input <-chan struct{}, delay time.Duration) <-chan struct{} {
	output := make(chan struct{})
	go func() {
		defer close(output)
		var timer *time.Timer
		var timerC <-chan time.Time
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-input:
				if !ok {
					return
				}
				if timer == nil {
					timer = time.NewTimer(delay)
				} else {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(delay)
				}
				timerC = timer.C
			case <-timerC:
				timerC = nil
				select {
				case output <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return output
}
