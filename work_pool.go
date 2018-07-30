package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	cc "airman.com/logstatd/pkg/cache"
	kc "airman.com/logstatd/pkg/kafka"
	st "airman.com/logstatd/pkg/stat"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	PrefixQueue    = "jobq"
	PrefixRecovery = "jobc"
)

type JobWork struct {
	Topic   string
	Msg     string
	IsCache bool `json:"is_cache"`
	Jid     uint64
}

type WorkerPool struct {
	num      int
	jobq     chan JobWork
	wg       sync.WaitGroup
	interval time.Duration
	wkctx    *contexts
	mkday    int
}

type contexts struct {
	ctxShutdown context.Context
	shutdown    context.CancelFunc
}

func newContext(parent context.Context) *contexts {
	ctxShutdown, shutdown := context.WithCancel(parent)
	return &contexts{
		ctxShutdown: ctxShutdown,
		shutdown:    shutdown,
	}
}

func NewWorkerPool(num, qsize, interval int) *WorkerPool {
	ctx := newContext(context.Background())
	return &WorkerPool{
		num:      num,
		jobq:     make(chan JobWork, qsize),
		interval: time.Duration(interval) * time.Millisecond,
		wkctx:    ctx,
		mkday:    time.Now().Day(),
	}
}

func (p *WorkerPool) Run(cache *cc.CacheData, cp *kc.ClientPool) {
	// run recovery worker
	go p.recoveryWorker(p.wkctx, 0, cache, cp)
	p.wg.Add(1)

	// run workers
	for i := 1; i < p.num+1; i++ {
		go p.worker(p.wkctx, i, cache, cp)
		p.wg.Add(1)
	}
	st.GetStat().SetStatus(st.STATUS_RUNNING)

}

func (p *WorkerPool) Stop() {
	p.wkctx.shutdown()
	p.wg.Wait()
	st.GetStat().SetStatus(st.STATUS_STOP)
}

func (p *WorkerPool) worker(ctx *contexts, idx int, cache *cc.CacheData, cp *kc.ClientPool) {
	defer p.wg.Done()

	log.Infof("worker index[%d] start", idx)
	for {
		isStop := func() bool {
			defer func() bool {
				if err := recover(); err != nil {
					// w.cli.Close()
					log.Errorf("panic:%v", err)
					for skip := 2; ; skip++ {
						pc, f, n, ok := runtime.Caller(skip)
						if !ok {
							break
						}
						log.Errorf("%v():%s:%d",
							filepath.Base(runtime.FuncForPC(pc).Name()), f, n)
					}
					return false
				}
				return false
			}()

			// get client from pool
			cli, err := cp.Get()
			if err != nil {
				log.Errorf("Get client error:%v", err)
				time.Sleep(p.interval)
				return false
			}
			defer cli.Close()

			//			var w *JobWork
			for {
				select {
				case w := <-p.jobq:
					// handle
					nw := &JobWork{
						Topic:   w.Topic,
						Msg:     w.Msg,
						IsCache: w.IsCache,
						Jid:     w.Jid,
					}
					err := mainHandle(nw, cache, cli)
					if err != nil {
						log.Errorf("handle error, %s:%s:%v:%v", nw.Topic, nw.Msg, nw.IsCache, err)
						cli.MarkErr()
						return false
					}
				case <-ctx.ctxShutdown.Done():
					log.Warnf("ctx done, worker index[%d] stop", idx)
					return true
				default:
					time.Sleep(p.interval)
				}
			}
		}()

		if isStop {
			break
		}
	}
	log.Warnf("worker index[%d] stop", idx)
}

func (p *WorkerPool) recoveryWorker(ctx *contexts, idx int, cache *cc.CacheData, cp *kc.ClientPool) {
	defer p.wg.Done()

	log.Infof("worker index[%d] start", idx)
	for {
		isStop := func() bool {
			defer func() bool {
				if err := recover(); err != nil {
					// w.cli.Close()
					log.Errorf("panic:%v", err)
					for skip := 2; ; skip++ {
						pc, f, n, ok := runtime.Caller(skip)
						if !ok {
							break
						}
						log.Errorf("%v():%s:%d",
							filepath.Base(runtime.FuncForPC(pc).Name()), f, n)
					}
					return false
				}
				return false
			}()

			// get client from pool
			cli, err := cp.Get()
			if err != nil {
				log.Errorf("Get client error:%v", err)
				time.Sleep(p.interval * 5)
				return false
			}
			defer cli.Close()

			var count int64
			for {
				if st.GetStat().GetStatus() == st.STATUS_RUNNING {
					iter := cache.DB.NewIterator(util.BytesPrefix([]byte(PrefixRecovery)), nil)
					for iter.Next() {
						log.Infof("recovery get k:%q, v:%q", iter.Key(), iter.Value())
						jobdata := &JobWork{}
						if err := json.Unmarshal(iter.Value(), jobdata); err != nil {
							log.Errorf("ParseJson from cache error(%v:%v)", string(iter.Key()), err)
							continue
						}

						// handle
						if err := RecoveryHandle(jobdata, cache, cli); err != nil {
							log.Errorf("RecoveryHandle error:%v", err)
							iter.Release()
							cli.MarkErr()
							return false
						}
						count = count + 1
						if count%1000 == 0 {
							time.Sleep(p.interval)
						}
					}
					iter.Release()
					err := iter.Error()
					if count != 0 {
						log.Infof("handle count:%d, error:%v", count, err)
					}
				}

				if count == 0 {
					if time.Now().Day() != p.mkday {
						st.GetStat().WriteStat()
						st.GetStat().ReSet()
						r.ReSetGenId()
						p.mkday = time.Now().Day()
					}
					time.Sleep(p.interval * 5)
				} else {
					time.Sleep(p.interval)
				}
				count = 0

				select {
				case <-ctx.ctxShutdown.Done():
					log.Warnf("ctx done, recovery worker index[%d] stop", idx)
					return true
				default:
					time.Sleep(p.interval)
				}
			}
		}()

		if isStop {
			break
		}
	}
	log.Warnf("worker index[%d] stop", idx)
}

func mainHandle(w *JobWork, cache *cc.CacheData, cli *kc.WrapClient) error {
	if err := cli.SendMsgByByte(w.Topic, w.Msg); err != nil {
		st.GetStat().Inc(st.ST_FAILED)
		if w.IsCache {
			dataval, err := json.Marshal(w)
			if err != nil {
				log.Errorf("mainHandle json error:%v,data(%v)", err, w)
			}

			if e := cache.Put([]byte(fmt.Sprintf("%s:%s:%v", PrefixRecovery, w.Topic, w.Jid)), dataval); e != nil {
				log.Errorf("Put recovery data, key:(%v:%v:%v) error:%v", PrefixRecovery, w.Topic, w.Jid, e)
				return e
			}
			st.GetStat().Inc(st.ST_RECOVERY)

			if e := cache.Delete([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, w.Topic, w.Jid))); e != nil {
				log.Errorf("Delete cache data, key:(%v:%v) error:%v", w.Topic, w.Jid, e)

				st.GetStat().Inc(st.ST_DELERR)
				return nil
			}
			log.Warnf("job send failed, add recovery key:(%s:%s:%v)", PrefixRecovery, w.Topic, w.Jid)
		}
		return err
	}

	st.GetStat().Inc(st.ST_SUCC)
	if w.IsCache {
		if e := cache.Delete([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, w.Topic, w.Jid))); e != nil {
			log.Errorf("Delete cache data, key:(%v:%v) error:%v", w.Topic, w.Jid, e)
			st.GetStat().Inc(st.ST_DELERR)
			//			return nil
		}
	}
	return nil
}

func RecoveryHandle(w *JobWork, cache *cc.CacheData, cli *kc.WrapClient) error {
	if err := cli.SendMsgByByte(w.Topic, w.Msg); err != nil {
		return err
	}

	st.GetStat().Inc(st.ST_RECOVERY_SUCC)
	log.Infof("delete key:%s::%s:%v", PrefixRecovery, w.Topic, w.Jid)
	if e := cache.Delete([]byte(fmt.Sprintf("%s:%s:%v", PrefixRecovery, w.Topic, w.Jid))); e != nil {
		st.GetStat().Inc(st.ST_DELERR)
		log.Errorf("Delete recovery data, key:(%v:%v) error:%v", PrefixRecovery, w.Jid, e)
		// return e
	}
	return nil
}

func (p *WorkerPool) InitRecoveryAll(cache *cc.CacheData) error {
	var count int64
	var w JobWork
	iter := cache.DB.NewIterator(nil, nil)
	for iter.Next() {
		if strings.HasPrefix(string(iter.Key()), PrefixQueue) {
			if err := json.Unmarshal(iter.Value(), &w); err != nil {
				log.Errorf("InitRecovery():ParseJson from cache error(%v:%v)", string(iter.Key()), err)
				continue
			}

			if e := cache.Put([]byte(fmt.Sprintf("%s:%s:%v", PrefixRecovery, w.Topic, w.Jid)), iter.Value()); e != nil {
				log.Errorf("InitRecovery():Put recovery data, key:(%v:%v:%v) error:%v", PrefixRecovery, w.Topic, w.Jid, e)
				continue
			}

			st.GetStat().Inc(st.ST_RECOVERY_INIT)

			if e := cache.Delete(iter.Key()); e != nil {
				log.Errorf("InitRecovery():Delete cache data, key:(%v:%v) error:%v", w.Topic, w.Jid, e)
				continue
			}
			count = count + 1
		}
	}
	iter.Release()
	if count != 0 {
		log.Infof("InitRecovery():handle count:%d", count)
	}
	if err := iter.Error(); err != nil {
		log.Errorf("InitRecovery(): handle err:%v", err)
		return err
	}
	return nil
}
