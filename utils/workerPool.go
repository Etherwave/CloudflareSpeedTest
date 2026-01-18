package utils

import (
	"context"
	"sync"
)

type Worker struct {
	workerPool *WorkerPool
}

func NewWorker(workerPool *WorkerPool) *Worker {
	return &Worker{
		workerPool: workerPool,
	}
}

func (w *Worker) run(ctx context.Context) {
	defer w.workerPool.wg.Done()
	for {
		select {
		case task, ok := <-w.workerPool.tasksChan:
			if !ok { // 通道关闭时退出
				return
			}
			if task != nil { // 防御 nil task
				task()
			}
		case <-ctx.Done():
			return
		}
	}
}

type WorkerPool struct {
	workersNum        int
	workers           []*Worker
	tasksChan         chan func()
	taskChanCloseOnce sync.Once
	ctx               context.Context
	cancel            context.CancelFunc
	cancelOnce        sync.Once
	wg                sync.WaitGroup
}

func NewWorkerPool(workersNum int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		workersNum: workersNum,
		workers:    make([]*Worker, workersNum),
		tasksChan:  make(chan func(), workersNum*2),
		ctx:        ctx,
		cancel:     cancel,
	}
	pool.wg.Add(workersNum)
	for i := 0; i < workersNum; i++ {
		pool.workers[i] = NewWorker(pool)
		go pool.workers[i].run(ctx)
	}
	return pool
}

func (pool *WorkerPool) Submit(task func()) {
	pool.tasksChan <- task
}

func (pool *WorkerPool) Wait() {
	pool.taskChanCloseOnce.Do(func() {
		close(pool.tasksChan)
	})
	pool.wg.Wait()
}

func (pool *WorkerPool) Stop() {
	pool.cancelOnce.Do(func() {
		pool.cancel() // 先通知退出
	})
	pool.taskChanCloseOnce.Do(func() {
		close(pool.tasksChan)
	})
	pool.wg.Wait() // 等待全部 goroutine 退出
}
