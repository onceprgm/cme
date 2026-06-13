package download

import (
	"errors"
	"sync"
	"sync/atomic"
)

type Task struct {
	URL  string
	Dest string
	SHA1 string
}

func All(tasks []Task, workers int, progress func(done, total int)) error {
	if workers < 1 {
		workers = 1
	}
	total := len(tasks)
	if total == 0 {
		return nil
	}

	queue := make(chan Task)
	errs := make([]error, total)
	var done atomic.Int64
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range queue {
				err := File(t.URL, t.Dest, t.SHA1)
				i := done.Add(1)
				if err != nil {
					errs[i-1] = err
				}
				if progress != nil {
					progress(int(i), total)
				}
			}
		}()
	}

	for _, t := range tasks {
		queue <- t
	}
	close(queue)
	wg.Wait()

	return errors.Join(errs...)
}
