package genid

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/zheng-ji/goSnowFlake"
)

type UidWorker struct {
	worker *goSnowFlake.IdWorker
}

func NewUidWorker(mid int64) (*UidWorker, error) {
	idworker, err := goSnowFlake.NewIdWorker(mid)
	if err != nil {
		log.Errorf("goSnowFlake NewIdWorker error:%v", err)
		return nil, err
	}
	return &UidWorker{idworker}, nil
}

func (u *UidWorker) GetId() (int64, error) {
	return u.worker.NextId()
}

type UidGen struct {
	gid  uint64
	lock sync.Mutex
}

func NewUidGen(beginNum uint64) *UidGen {
	return &UidGen{gid: beginNum}
}

func (u *UidGen) GetGenId() uint64 {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.gid++
	return u.gid
}

func (u *UidGen) ReSetGenId() {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.gid = 0
}
