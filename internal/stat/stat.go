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
	// 0-init, 1-running, 2-stop, 3-recovery, 4-fault
	Status             STATUS_T  `json:"status"`
	UpTime             time.Time `json:"up_time"`
	JobNum             uint64    `json:"job_num"`
	ReqNum             uint64    `json:"req_num"`
	SuccessNum         uint64    `json:"succ_num"`
	ErrorNum           uint64    `json:"err_num"`
	CacheNum           uint64    `json:"cache_num"`
	DelErrNum          uint64    `json:"del_err_num"`
	RecoveryNum        uint64    `json:"recov_num"`
	RecoverySuccessNum uint64    `json:"recov_succ_num"`
	RecoveryInitNum    uint64    `json:"recov_init_num"`
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
		atomic.AddUint64(&s.SuccessNum, 1)
	case ST_FAILED:
		atomic.AddUint64(&s.ErrorNum, 1)
	case ST_DELERR:
		atomic.AddUint64(&s.DelErrNum, 1)
	case ST_RECOVERY_SUCC:
		atomic.AddUint64(&s.RecoverySuccessNum, 1)
	case ST_RECOVERY:
		atomic.AddUint64(&s.RecoveryNum, 1)
	case ST_RECOVERY_INIT:
		atomic.AddUint64(&s.RecoveryInitNum, 1)
	}
}

func (s *Stat) ReSet() {
	atomic.StoreUint64(&s.JobNum, 0)
	atomic.StoreUint64(&s.ReqNum, 0)
	atomic.StoreUint64(&s.SuccessNum, 0)
	atomic.StoreUint64(&s.ErrorNum, 0)
	atomic.StoreUint64(&s.DelErrNum, 0)
	atomic.StoreUint64(&s.RecoverySuccessNum, 0)
	atomic.StoreUint64(&s.RecoveryNum, 0)
	atomic.StoreUint64(&s.RecoveryInitNum, 0)
}

func (s *Stat) NowStat() string {
	return fmt.Sprintf("stats >>\n jobq number:%d\n req number:%d\n succ number:%d \n failed number:%d\n delete_err number:%d\n recovery number:%d\n recovery init number:%d\n recovery success number:%d\n",
		s.JobNum, s.ReqNum, s.SuccessNum, s.ErrorNum, s.DelErrNum, s.RecoveryNum, s.RecoveryInitNum, s.RecoverySuccessNum)
}

func (s *Stat) WriteStat() {
	log.Warnf("stats >>\n jobq number:%d\n req number:%d\n succ number:%d \n failed number:%d\n delete_err number:%d\n recovery number:%d\n recovery init number:%d\n recovery success number:%d\n",
		s.JobNum, s.ReqNum, s.SuccessNum, s.ErrorNum, s.DelErrNum, s.RecoveryNum, s.RecoveryInitNum, s.RecoverySuccessNum)

}
