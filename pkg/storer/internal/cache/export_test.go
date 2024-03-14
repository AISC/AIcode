// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"fmt"
	"time"

	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/storer/internal"
	"github.com/aisc/pkg/ aisc"
)

type (
	CacheEntry = cacheEntry
)

var (
	ErrMarshalCacheEntryInvalidAddress   = errMarshalCacheEntryInvalidAddress
	ErrMarshalCacheEntryInvalidTimestamp = errMarshalCacheEntryInvalidTimestamp
	ErrUnmarshalCacheEntryInvalidSize    = errUnmarshalCacheEntryInvalidSize
)

func ReplaceTimeNow(fn func() time.Time) func() {
	now = fn
	return func() {
		now = time.Now
	}
}

type CacheState struct {
	Head  aisc.Address
	Tail  aisc.Address
	Size uint64
}

func (c *Cache) RemoveOldestMaxBatch(ctx context.Context, store internal.Storage, chStore storage.ChunkStore, count uint64, batchCnt int) error {
	return c.removeOldest(ctx, store, store.ChunkStore(), count, batchCnt)
}

func (c *Cache) State(store storage.Store) CacheState {
	state := CacheState{}
	state.Size = c.Size()
	runner :=  aisc.ZeroAddress

	err := store.Iterate(
		storage.Query{
			Factory:      func() storage.Item { return &cacheOrderIndex{} },
			ItemProperty: storage.QueryItemID,
		},
		func(res storage.Result) (bool, error) {
			_, addr, err := idFromKey(res.ID)
			if err != nil {
				return false, err
			}

			if state.Head.Equal( aisc.ZeroAddress) {
				state.Head = addr
			}
			runner = addr
			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}
	state.Tail = runner
	return state
}

func (c *Cache) IterateOldToNew(
	st storage.Store,
	start, end  aisc.Address,
	iterateFn func(ch  aisc.Address) (bool, error),
) error {
	runner :=  aisc.ZeroAddress
	err := st.Iterate(
		storage.Query{
			Factory:      func() storage.Item { return &cacheOrderIndex{} },
			ItemProperty: storage.QueryItemID,
		},
		func(res storage.Result) (bool, error) {
			_, addr, err := idFromKey(res.ID)
			if err != nil {
				return false, err
			}

			if runner.Equal( aisc.ZeroAddress) {
				if !addr.Equal(start) {
					return false, fmt.Errorf("invalid cache order index key %s", res.ID)
				}
			}
			runner = addr
			return iterateFn(runner)
		},
	)
	if err != nil {
		return err
	}
	if !runner.Equal(end) {
		return fmt.Errorf("invalid cache order index key %s", runner.String())
	}

	return nil
}
