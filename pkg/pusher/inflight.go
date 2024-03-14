// Copyright 2021 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pusher

import (
	"sync"

	"github.com/aisc/pkg/ aisc"
)

type inflight struct {
	mtx      sync.Mutex
	inflight map[string]struct{}
}

func newInflight() *inflight {
	return &inflight{
		inflight: make(map[string]struct{}),
	}
}

func (i *inflight) delete(ch  aisc.Chunk) {
	key := ch.Address().ByteString() + string(ch.Stamp().BatchID())
	i.mtx.Lock()
	delete(i.inflight, key)
	i.mtx.Unlock()
}

func (i *inflight) set(ch  aisc.Chunk) bool {

	i.mtx.Lock()
	defer i.mtx.Unlock()

	key := ch.Address().ByteString() + string(ch.Stamp().BatchID())
	if _, ok := i.inflight[key]; ok {
		return true
	}

	i.inflight[key] = struct{}{}
	return false
}

type attempts struct {
	mtx        sync.Mutex
	retryCount int
	attempts   map[string]int
}

// try to log a chunk sync attempt. returns false when
// maximum amount of attempts have aiscn reached.
func (a *attempts) try(ch  aisc.Address) bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	key := ch.ByteString()
	a.attempts[key]++
	return a.attempts[key] < a.retryCount
}

func (a *attempts) delete(ch  aisc.Address) {
	a.mtx.Lock()
	delete(a.attempts, ch.ByteString())
	a.mtx.Unlock()
}
