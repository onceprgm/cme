package download

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

type Task struct {
	URL  string
	Dest string
	SHA1 string
}

func DefaultWorkers() int {
	n := runtime.NumCPU() * 2
	if n < 4 {
		n = 4
	}
	if n > 16 {
		n = 16
	}
	return n
}

func All(tasks []Task, workers int, progress func(done, total int)) error {
	if workers < 1 {
		workers = DefaultWorkers()
	}
	total := len(tasks)
	if total == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type indexed struct {
		task Task
		i    int
	}
	queue := make(chan indexed, workers*2)
	errs := make([]error, total)
	var done atomic.Int64
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range queue {
				if ctx.Err() != nil {
					return
				}
				if err := File(it.task.URL, it.task.Dest, it.task.SHA1); err != nil {
					errs[it.i] = err
					cancel()
				}
				if progress != nil {
					progress(int(done.Add(1)), total)
				}
			}
		}()
	}

	for i, t := range tasks {
		select {
		case queue <- indexed{task: t, i: i}:
		case <-ctx.Done():
		}
	}
	close(queue)
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
