package service

import (
	"apiChecker/internal/dto"
	"sync"

	"context"
)

type WorkerPool struct {
	maxWorkers int
	checker    *Checker
	inChan     <-chan dto.ResourceItem
	outChan    chan<- dto.CheckItem
}

func NewWorkerPool(
	maxWorkers int,
	checker *Checker,
	inChan <-chan dto.ResourceItem,
	outChan chan<- dto.CheckItem,
) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	return &WorkerPool{
		maxWorkers: maxWorkers,
		checker:    checker,
		inChan:     inChan,
		outChan:    outChan,
	}
}

func (wp *WorkerPool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case res, ok := <-wp.inChan:
			if !ok {
				return
			}

			checkResult := wp.checker.Check(ctx, res)
			if checkResult == nil {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case wp.outChan <- *checkResult:
			}
		}
	}
}

func (wp *WorkerPool) RunWorkerPool(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < wp.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wp.worker(ctx)
		}()
	}

	wg.Wait()

	close(wp.outChan)
}
