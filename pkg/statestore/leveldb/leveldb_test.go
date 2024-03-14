// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package leveldb_test

import (
	"testing"

	"github.com/aisc/pkg/log"
	"github.com/aisc/pkg/statestore/leveldb"
	"github.com/aisc/pkg/statestore/test"
	"github.com/aisc/pkg/storage"
)

func TestPersistentStateStore(t *testing.T) {
	test.Run(t, func(t *testing.T) storage.StateStorer {
		t.Helper()

		dir := t.TempDir()

		store, err := leveldb.NewStateStore(dir, log.Noop)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
		})

		return store
	})

	test.RunPersist(t, func(t *testing.T, dir string) storage.StateStorer {
		t.Helper()

		store, err := leveldb.NewStateStore(dir, log.Noop)
		if err != nil {
			t.Fatal(err)
		}

		return store
	})
}
