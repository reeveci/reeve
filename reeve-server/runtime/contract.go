package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type cancelCbFunc func(string)

type Contract struct {
	sync.Mutex
	Consumer sync.Mutex
	Contract string
	cancelCb cancelCbFunc
	cancel   context.CancelFunc
}

func (c *Contract) Next(timeout time.Duration, cancelCb cancelCbFunc) string {
	c.Consumer.Lock()
	defer c.Consumer.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	contract := uuid.NewString()
	c.Contract = contract
	c.cancelCb = cancelCb
	var ctx context.Context
	ctx, c.cancel = context.WithTimeout(context.Background(), timeout)

	go func() {
		<-ctx.Done()
		c.Consumer.Lock()
		defer c.Consumer.Unlock()

		if c.Contract == contract && c.cancel != nil {
			c.cancel = nil
			c.Unlock()
			if c.cancelCb != nil {
				c.cancelCb(contract)
			}
		}
	}()

	return c.Contract
}

func (c *Contract) Cancel() {
	c.Consumer.Lock()
	defer c.Consumer.Unlock()

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
		c.Unlock()
	}
}

func (c *Contract) IsCanceled() bool {
	return c.cancel == nil
}

func (c *Contract) Finish() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
		c.Unlock()
	}
	c.Consumer.Unlock()
}
