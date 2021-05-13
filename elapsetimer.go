package main

import (
	"sync/atomic"
	"time"
)

type watchdogTimer struct {
	epoch int64
}

func (tm *watchdogTimer) Update() {
	now := time.Now().Unix()
	atomic.StoreInt64(&(tm.epoch), now)
}

func (tm *watchdogTimer) IsElapsed(interval time.Duration) bool {
	et := atomic.LoadInt64(&(tm.epoch))
	now := time.Now().Unix()

	return now > (et + int64(interval.Seconds()))
}
