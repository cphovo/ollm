package util

import (
	"sync"
	"time"
)

type item struct {
	value      any
	expiryTime *time.Time
}

type LazyExpiryKVStore struct {
	store map[string]item
	mutex sync.RWMutex
}

func NewLazyExpiryStore() *LazyExpiryKVStore {
	return &LazyExpiryKVStore{
		store: make(map[string]item),
	}
}

func (kv *LazyExpiryKVStore) Set(key string, value any, expiryInSeconds int) {
	kv.mutex.Lock()
	defer kv.mutex.Unlock()

	var expiryTime *time.Time
	if expiryInSeconds > 0 {
		t := time.Now().Add(time.Duration(expiryInSeconds) * time.Second)
		expiryTime = &t
	}

	kv.store[key] = item{value: value, expiryTime: expiryTime}
}

func (kv *LazyExpiryKVStore) Get(key string) (any, bool) {
	kv.mutex.RLock()
	defer kv.mutex.RUnlock()

	if item, ok := kv.store[key]; ok {
		if item.expiryTime == nil || item.expiryTime.After(time.Now()) {
			return item.value, true
		}

		// 如果键已过期，删除
		kv.Delete(key)
	}
	return nil, false
}

func (kv *LazyExpiryKVStore) Delete(key string) {
	kv.mutex.Lock()
	defer kv.mutex.Unlock()

	delete(kv.store, key)
}
