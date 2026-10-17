package ops

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConcurrentCommandsOnOneSpecAllLand(t *testing.T) {
	root := startedStore(t)
	spec := specAt(root, "started", "s")
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- Log(root, spec, fmt.Sprintf("note %d", i), now)
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got := readSpec(t, root)
	for i := 0; i < 8; i++ {
		if !strings.Contains(got, fmt.Sprintf("note %d\n", i)) {
			t.Errorf("note %d lost:\n%s", i, got)
		}
	}
}

func TestCommandsWaitForTheStoreLock(t *testing.T) {
	root := startedStore(t)
	release := Store{Root: root}.lock()
	done := make(chan error, 1)
	go func() { done <- Log(root, specAt(root, "started", "s"), "waits", now) }()
	select {
	case <-done:
		t.Fatal("Log ran while the lock was held")
	case <-time.After(300 * time.Millisecond):
	}
	release()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
