package genid

import (
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
