package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/yuleihua/logstatd/internal/cache"
	"github.com/yuleihua/logstatd/internal/conf"
	gid "github.com/yuleihua/logstatd/internal/genid"
	"github.com/yuleihua/logstatd/internal/kafka"
	"github.com/yuleihua/logstatd/internal/stat"
)

// var r = rand.New(rand.NewSource(time.Now().UnixNano()))
// const INT_MAX int64 = int64(^uint64(0) >> 1)

func GetMachineId(address string) int64 {
	var temp string
	ipList := strings.Split(address, ":")
	if len(ipList) <= 1 || ipList[0] == "" || ipList[0] == "*" {
		temp = os.Getenv("ENV_MID")
	} else {
		temp = strings.Split(ipList[0], ".")[3]
	}

	mid, err := strconv.ParseInt(temp, 10, 64)
	if err != nil {
		log.Fatalf("invalid ip string:%v", temp)
	}
	return mid
}

type Server struct {
	topic   string
	wp      *WorkerPool
	cache   *cache.CacheData
	cp      *kafka.ClientPool
	uw      *gid.UidWorker
	ErrChan chan error
	mode    int
}

func NewServer(sc *conf.Service, sp *conf.Kafka) (*Server, error) {
	if sc == nil || sp == nil {
		return nil, errors.New("invalid parameter")
	}

	var err error
	s := &Server{
		ErrChan: make(chan error, 16),
	}

	s.uw, err = gid.NewUidWorker(GetMachineId(sc.Address))
	if err != nil {
		log.Errorf("uid init error:%v", err)
		return nil, err
	}

	log.Info("server initiation begin")
	log.Info("server initiation step1: create kafka client pool")

	if s.cp == nil {
		fn := func() (*kafka.KafkaClient, error) {
			return kafka.NewKafkaClient(sp.Address, time.Duration(sp.Timeout)*time.Second, true)
		}
		s.cp, err = kafka.NewClientPool(sc.PoolMin, sc.PoolMax, fn)
		if err != nil {
			return nil, err
		}
	}

	log.Info("server initiation step2: create leveldb cache")
	if s.cache == nil {
		s.cache = cache.NewCacheData(sc.DBName)
	}

	log.Info("server initiation step3: create worker pool")
	// pool
	if s.wp == nil {
		s.wp = NewWorkerPool(sc.WorkerNum, sc.JobNum, sc.Interval)
		s.wp.Run(s.cache, s.cp)
		if err := s.wp.InitRecoveryAll(s.cache); err != nil {
			return nil, err
		}
	}

	s.topic = sp.Address

	log.Info("server initiation end")
	return s, nil
}

func (s *Server) Shutdown() error {
	log.Warn("server shutdown begin")

	log.Warn("server shutdown step1: shutdown worker pool")
	if s.wp != nil {
		s.wp.Stop()
	}

	log.Warn("server shutdown step2: shutdown leveldb cache")
	if s.cp != nil {
		s.cp.Close()
	}

	log.Warn("server shutdown step3: shutdown kafka client pool")
	if s.cache != nil {
		s.cache.Close()
	}
	log.Warn("server shutdown end")
	return nil
}

func (s *Server) TransportMsg(topic, msg string, isWeb bool) {
	if s.wp != nil && isWeb {
		uuid, err := s.GetUUid()
		if err != nil {
			log.Errorf("GetUid error:%v,data(%v)", err, uuid)
			return
		}
		obj := JobWork{topic, msg, false, uint64(uuid)}

		content, err := json.Marshal(obj)
		if err != nil {
			log.Errorf("json error:%v,data(%v)", err, obj)
			return
		}

		if err := s.cache.Put([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, obj.Topic, obj.Jid)), content); err != nil {
			log.Errorf("cache put error:%v,data(%v)", err, obj)

			isExisted, err := s.cache.Has([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, obj.Topic, obj.Jid)))
			if err != nil {
				log.Errorf("cache has error:%v,data(%v)", err, obj)
			}

			if isExisted {
				uid, err := s.GetUUid()
				if err != nil {
					log.Errorf("GetUid error:%v,data(%v)", err, obj)
					return
				}
				obj.Jid = uint64(uid)

				content, err := json.Marshal(obj)
				if err != nil {
					log.Errorf("json error:%v,data(%v)", err, obj)
					return
				}

				if err := s.cache.Put([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, obj.Topic, obj.Jid)), content); err != nil {
					log.Errorf("cache put error:%v,data(%v)", err, obj)
					return
				}
			} else {
				log.Warnf("log item is not existed, but put cache error, %v, %v", err, obj)
			}
		}

		select {
		case s.wp.jobq <- obj:
			stat.GetStat().Inc(stat.ST_JOBQ)
		default:
			// add recovery when jobq is full
			obj.IsCache = true
			if e := s.cache.Put([]byte(fmt.Sprintf("%s:%s:%v", PrefixRecovery, obj.Topic, obj.Jid)), content); e != nil {
				log.Errorf("jobq is full, Put recovery data, key:(%v:%v:%v) error:%v", PrefixRecovery, obj.Topic, obj.Jid, e)
				return
			}
			stat.GetStat().Inc(stat.ST_RECOVERY)

			if e := s.cache.Delete([]byte(fmt.Sprintf("%s:%s:%v", PrefixQueue, obj.Topic, obj.Jid))); e != nil {
				log.Errorf("jobq is full, Delete cache data, key:(%v:%v) error:%v", obj.Topic, obj.Jid, e)
				stat.GetStat().Inc(stat.ST_DELERR)
			}
			log.Warnf("jobq is full, add recovery key:(%s:%s:%v)", PrefixRecovery, obj.Topic, obj.Jid)
		}
	}
}

func (s *Server) GetUUid() (int64, error) {
	if s.uw != nil {
		uid, err := s.uw.GetId()
		if err != nil {
			log.Errorf("GetId error:%v", err)
			return 0, err
		} else {
			return uid, nil
		}
	}
	return 0, errors.New("internal error")
}
