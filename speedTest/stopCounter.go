package speedTest

import (
	"sync"
	"sync/atomic"
)

type StopCounter struct {
	target int64
	count  int64
	once   sync.Once
	ch     chan struct{}
}

func NewStopCounter(target int) *StopCounter {
	return &StopCounter{target: int64(target), ch: make(chan struct{})}
}

// Inc 成功一次返回 true，表示"刚达到目标"
func (c *StopCounter) Inc() bool {
	if atomic.AddInt64(&c.count, 1) >= c.target {
		c.once.Do(func() { close(c.ch) })
		return true
	}
	return false
}

func (c *StopCounter) Done() <-chan struct{} { return c.ch }

func (c *StopCounter) Stopped() bool {
	select {
	case <-c.ch:
		return true
	default:
		return false
	}
}
