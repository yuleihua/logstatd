package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	cc "airman.com/logstatd/pkg/cache"
	"airman.com/logstatd/pkg/conf"
	gid "airman.com/logstatd/pkg/genid"
	kc "airman.com/logstatd/pkg/kafka"
	st "airman.com/logstatd/pkg/stat"
	log "github.com/sirupsen/logrus"
)

var r = gid.NewUidGen(100000)

// var r = rand.New(rand.NewSource(time.Now().UnixNano()))
// const INT_MAX int64 = int64(^uint64(0) >> 1)

type Server struct {
	wp    *WorkerPool
	cache *cc.CacheData
	cp    *kc.ClientPool
	uw    *gid.UidWorker
	mode  int
}

var _ss *Server

func GetServer() *Server {
	return _ss
}

func NewServer(sc *conf.Service, sp *conf.Proxy) error {
	if sc == nil || sp == nil {
		return errors.New("invalid parameter")
	}

	var err error
	var srv Server
	srv.uw, err = gid.NewUidWorker(getMachineId(sc.SrvAddr))
	if err != nil {
		log.Errorf("uid init error:%v", err)
		return err
	}

	log.Infof("server initiation begin")
	log.Infof("server initiation step1: create kafka client pool")

	if srv.cp == nil {
		fn := func() (*kc.KafkaClient, error) {
			return kc.NewKafkaClient(sp.Proxy, time.Duration(sp.Timeout)*time.Second, true)
		}
		srv.cp, err = kc.NewClientPool(sc.PoolMin, sc.PoolMax, fn)
		if err != nil {
			return err
		}
	}

	log.Infof("server initiation step2: create leveldb cache")
	if srv.cache == nil {
		srv.cache = cc.NewCacheData(sc.FileName)
	}

	log.Infof("server initiation step3: create worker pool")
	// pool
	if srv.wp == nil {
		srv.wp = NewWorkerPool(sc.WorkerNum, sc.JobqNum, sc.Interval)
		srv.wp.Run(srv.cache, srv.cp)
		if err := srv.wp.InitRecoveryAll(srv.cache); err != nil {
			return err
		}
	}
	log.Infof("server initiation end")
	_ss = &srv
	return nil
}

func (srv *Server) Shutdown() error {
	log.Warnf("server shutdown begin")

	log.Warnf("server shutdown step1: shutdown worker pool")
	if srv.wp != nil {
		srv.wp.Stop()
	}

	log.Warnf("server shutdown step2: shutdown leveldb cache")
	if srv.cp != nil {
		srv.cp.Close()
	}

	log.Warnf("server shutdown step3: shutdown kafka client pool")
	if srv.cache != nil {
		srv.cache.Close()
	}
	log.Warnf("server shutdown end")
	return nil
}

func (srv *Server) TransportMsg(topic, msg string, Jid uint64, isWeb bool) {
	if srv.wp != nil && isWeb {
		obj := JobWork{topic, msg, false, Jid}

		obj.Jid = r.GetGenId()
		isExisted, err := srv.cache.Has([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, obj.Topic, obj.Jid)))
		if err != nil {
			log.Errorf("cache has error:%v,data(%v)", err, obj)
		}
		if isExisted {
			uid, err := srv.GetUid()
			if err != nil {
				log.Errorf("GetUid error:%v,data(%v)", err, obj)
				return
			}
			obj.Jid = uint64(uid)
		}
		dataval, err := json.Marshal(obj)
		if err != nil {
			log.Errorf("json error:%v,data(%v)", err, obj)
			return
		}

		err = srv.cache.Put([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, obj.Topic, obj.Jid)), dataval)
		if err != nil {
			log.Errorf("cache put error:%v,data(%v)", err, obj)
		}

		select {
		case srv.wp.jobq <- obj:
			st.GetStat().Inc(st.ST_JOBQ)
		default:
			// add recovery when jobq is full
			obj.IsCache = true
			if e := srv.cache.Put([]byte(fmt.Sprintf("%s:%s:%v", PrefixRecovery, obj.Topic, obj.Jid)), dataval); e != nil {
				log.Errorf("jobq is full, Put recovery data, key:(%v:%v:%v) error:%v", PrefixRecovery, obj.Topic, obj.Jid, e)
				return
			}
			st.GetStat().Inc(st.ST_RECOVERY)

			if e := srv.cache.Delete([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, obj.Topic, obj.Jid))); e != nil {
				log.Errorf("jobq is full, Delete cache data, key:(%v:%v) error:%v", obj.Topic, obj.Jid, e)
				st.GetStat().Inc(st.ST_DELERR)
			}
			log.Warnf("jobq is full, add recovery key:(%s:%s:%v)", PrefixRecovery, obj.Topic, obj.Jid)
		}
	}
}

func (srv *Server) GetUid() (int64, error) {
	if srv.uw != nil {
		uid, err := srv.uw.GetId()
		if err != nil {
			log.Errorf("GetId error:%v", err)
			return 0, err
		} else {
			return uid, nil
		}
	}
	return 0, errors.New("internal error")
}
