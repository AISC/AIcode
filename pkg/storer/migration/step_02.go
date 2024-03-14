// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migration

import (
	"time"

	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/storer/internal/cache"
	"github.com/aisc/pkg/ aisc"
)

// step_02 migrates the cache to the new format.
// the old cacheEntry item has the same key, but the value is different. So only
// a Put is needed.
func step_02(st storage.BatchedStore) error {
	var entries []*cache.CacheEntryItem
	err := st.Iterate(
		storage.Query{
			Factory:      func() storage.Item { return &cache.CacheEntryItem{} },
			ItemProperty: storage.QueryItemID,
		},
		func(res storage.Result) (bool, error) {
			entry := &cache.CacheEntryItem{
				Address:          aisc.NewAddress([]byte(res.ID)),
				AccessTimestamp: time.Now().UnixNano(),
			}
			entries = append(entries, entry)
			return false, nil
		},
	)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		err := st.Put(entry)
		if err != nil {
			return err
		}
	}

	return nil
}
