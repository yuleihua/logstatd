package kafka

import (
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var ErrClosed = errors.New("pool is closed")

type ClientPool struct {
	mu      sync.Mutex
	clis    chan *WrapClient
	factory Factory
	cap     int
}

type Factory func() (*KafkaClient, error)

func NewClientPool(min, max int, factory Factory) (*ClientPool, error) {
	if min < 0 || max <= 0 {
		return nil, errors.New("invalid capacity settings")
	}

	if min > max {
		min = max
	}

	c := &ClientPool{
		clis:    make(chan *WrapClient, max),
		factory: factory,
		cap:     max,
	}

	for i := 0; i < min; i++ {
		cli, err := factory()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("running factory err: %s", err)
		}
		c.clis <- &WrapClient{
			KafkaClient: cli,
			cp:          c,
		}
		log.Infof("put new cli in pool:%d,%v", i, cli)
	}
	return c, nil
}

func (c *ClientPool) getClientsAndFactory() (chan *WrapClient, Factory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	clis := c.clis
	factory := c.factory
	return clis, factory
}

func (c *ClientPool) Get() (*WrapClient, error) {
	clis, factory := c.getClientsAndFactory()
	if clis == nil {
		return nil, ErrClosed
	}

	wr := &WrapClient{
		cp: c,
	}
	select {
	case wr = <-clis:
		if wr.KafkaClient == nil {
			log.Warnf("kafka client is nil")
		}
		log.Infof("get existed cli in pool:%v", wr.KafkaClient)
	default:
	}

	var err error
	if wr.KafkaClient == nil {
		wr.KafkaClient, err = factory()
		if err != nil {
			log.Errorf("kafka cliet new error:%v", err)
			return nil, err
		}
	}
	wr.useTime = time.Now()
	return wr, nil
}

func (c *ClientPool) put(cli *WrapClient) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.clis == nil || len(c.clis) == c.cap {
		log.Warnf("Close client directly, %v", cli.KafkaClient)
		return cli.KafkaClient.Close()
	}

	select {
	case c.clis <- cli:
		log.Infof("put free cli in pool:%v", cli.KafkaClient)
		return nil
	default:
		return cli.KafkaClient.Close()
	}
}

func (c *ClientPool) Close() {
	if c.clis == nil {
		return
	}

	c.mu.Lock()
	clis := c.clis
	c.clis = nil
	c.factory = nil
	c.mu.Unlock()

	close(clis)
	for cli := range clis {
		cli.KafkaClient.Close()
	}
}

func (c *ClientPool) Len() int {
	clis, _ := c.getClientsAndFactory()
	return len(clis)
}

//-----------------------------------------------------------------------------
type WrapClient struct {
	*KafkaClient
	mu      sync.RWMutex
	cp      *ClientPool
	isErr   bool
	useTime time.Time
}

func (wc *WrapClient) Close() error {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	log.Infof("client running time:%v", time.Now().Sub(wc.useTime).Seconds())
	if wc.isErr {
		if wc.KafkaClient != nil {
			return wc.KafkaClient.Close()
		}
		return nil
	}

	wcli := &WrapClient{
		KafkaClient: wc.KafkaClient,
		cp:          wc.cp,
	}
	return wc.cp.put(wcli)
}

func (p *WrapClient) MarkErr() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isErr = true
}
