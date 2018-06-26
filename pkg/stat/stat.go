package stat

import (
	"fmt"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

type ST_T int

const (
	ST_JOBQ ST_T = iota
	ST_REQ
	ST_SUCC
	ST_FAILED
	ST_DELERR
	ST_RECOVERY_INIT
	ST_RECOVERY_SUCC
	ST_RECOVERY
)

type STATUS_T uint8

const (
	STATUS_INIT STATUS_T = iota
	STATUS_RUNNING
	STATUS_STOP
)

type Stat struct {
	Status       STATUS_T // 0-init, 1-running, 2-stop, 3-recovery, 4-fault
	UpTime       time.Time
	JobNum       uint64
	ReqNum       uint64
	SuccNum      uint64
	ErrNum       uint64
	CacheNum     uint64
	DelErrNum    uint64
	RecovNum     uint64
	RecovSuccNum uint64
	RecovInitNum uint64
}

var _st = &Stat{}

func init() {
	_st.Status = STATUS_RUNNING
	_st.UpTime = time.Now()
}

func GetStat() *Stat {
	return _st
}

func (s *Stat) SetStatus(st STATUS_T) {
	s.Status = st
}

func (s *Stat) GetStatus() STATUS_T {
	return s.Status
}

func (s *Stat) Inc(t ST_T) {
	switch t {
	case ST_JOBQ:
		atomic.AddUint64(&s.JobNum, 1)
	case ST_REQ:
		atomic.AddUint64(&s.ReqNum, 1)
	case ST_SUCC:
		atomic.AddUint64(&s.SuccNum, 1)
	case ST_FAILED:
		atomic.AddUint64(&s.ErrNum, 1)
	case ST_DELERR:
		atomic.AddUint64(&s.DelErrNum, 1)
	case ST_RECOVERY_SUCC:
		atomic.AddUint64(&s.RecovSuccNum, 1)
	case ST_RECOVERY:
		atomic.AddUint64(&s.RecovNum, 1)
	case ST_RECOVERY_INIT:
		atomic.AddUint64(&s.RecovInitNum, 1)
	}
}

func (s *Stat) ReSet() {
	atomic.StoreUint64(&s.JobNum, 0)
	atomic.StoreUint64(&s.ReqNum, 0)
	atomic.StoreUint64(&s.SuccNum, 0)
	atomic.StoreUint64(&s.ErrNum, 0)
	atomic.StoreUint64(&s.DelErrNum, 0)
	atomic.StoreUint64(&s.RecovSuccNum, 0)
	atomic.StoreUint64(&s.RecovNum, 0)
	atomic.StoreUint64(&s.RecovInitNum, 0)
}

func (s *Stat) NowStat() string {
	return fmt.Sprintf("stats >>\n jobq number:%d\n req number:%d\n succ number:%d \n failed number:%d\n delete_err number:%d\n recovery number:%d\n recovery init number:%d\n recovery success number:%d\n",
		s.JobNum, s.ReqNum, s.SuccNum, s.ErrNum, s.DelErrNum, s.RecovNum, s.RecovInitNum, s.RecovSuccNum)
}

func (s *Stat) WriteStat() {
	log.Warnf("stats >>\n jobq number:%d\n req number:%d\n succ number:%d \n failed number:%d\n delete_err number:%d\n recovery number:%d\n recovery init number:%d\n recovery success number:%d\n",
		s.JobNum, s.ReqNum, s.SuccNum, s.ErrNum, s.DelErrNum, s.RecovNum, s.RecovInitNum, s.RecovSuccNum)

}
