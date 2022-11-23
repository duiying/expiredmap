package expiredmap

import (
	"go.uber.org/atomic"
	"sync"
	"time"
)

type ExpiredMap[TK comparable, TV any] struct {
	m              sync.Map
	expireInterval time.Duration
	capacity       int64
	len            atomic.Int64
	stop           chan bool
}

type expiredMapItem[TK comparable, TV any] struct {
	key      TK
	value    TV
	expireAt atomic.Time
}

func NewExpiredMap[TK comparable, TV any](capacity int64, expireInterval time.Duration) *ExpiredMap[TK, TV] {
	em := &ExpiredMap[TK, TV]{
		m:              sync.Map{},
		expireInterval: expireInterval,
		capacity:       capacity,
		stop:           make(chan bool),
	}
	em.start()
	return em
}

func (em *ExpiredMap[TK, TV]) start() {
	go func() {
		ticker := time.NewTicker(em.expireInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				em.m.Range(func(key, value any) bool {
					v, ok := value.(*expiredMapItem[TK, TV])
					if ok && v != nil && time.Now().After(v.expireAt.Load()) {
						TKey := key.(TK)
						em.Delete(TKey)
					}
					return true
				})
			case <-em.stop:
				return
			}
		}
	}()
}

func (em *ExpiredMap[TK, TV]) Set(key TK, value TV, ttl time.Duration) bool {
	if em.Size() >= em.capacity {
		return false
	}
	v := &expiredMapItem[TK, TV]{
		key:   key,
		value: value,
	}
	v.expireAt.Store(time.Now().Add(ttl))
	em.m.Store(key, v)
	em.len.Inc()
	return true
}

func (em *ExpiredMap[TK, TV]) Get(key TK) (value TV, ok bool) {
	var v any
	v, ok = em.m.Load(key)
	if !ok {
		return
	}
	v2, ok2 := v.(*expiredMapItem[TK, TV])
	if !ok2 || v2 == nil {
		return
	}

	if time.Now().After(v2.expireAt.Load()) {
		ok = false
		em.Delete(key)
		return
	}
	value = v2.value
	return
}

func (em *ExpiredMap[TK, TV]) Delete(key TK) {
	em.m.Delete(key)
	em.len.Dec()
}

func (em *ExpiredMap[TK, TV]) Size() int64 {
	return em.len.Load()
}

func (em *ExpiredMap[TK, TV]) TTL(key TK) time.Duration {
	v, ok := em.m.Load(key)
	if !ok {
		return -2
	}
	v2, ok2 := v.(*expiredMapItem[TK, TV])
	if !ok2 || v2 == nil {
		return -2
	}
	now := time.Now()
	if now.After(v2.expireAt.Load()) {
		em.Delete(key)
		return -1
	}
	return v2.expireAt.Load().Sub(now)
}

func (em *ExpiredMap[TK, TV]) HandleForeach(callback func(key TK, value TV)) {
	em.m.Range(func(key, value any) bool {
		v, ok := value.(*expiredMapItem[TK, TV])
		if !ok || v == nil {
			return false
		}
		if time.Now().After(v.expireAt.Load()) {
			TKey := key.(TK)
			em.Delete(TKey)
			return true
		}
		TKey := key.(TK)
		callback(TKey, v.value)
		return true
	})
}

func (em *ExpiredMap[TK, TV]) Close() {
	em.stop <- true
	em.m.Range(func(key, value any) bool {
		TKey := key.(TK)
		em.Delete(TKey)
		return true
	})
}
