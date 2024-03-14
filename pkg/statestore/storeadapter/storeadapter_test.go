// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package storeadapter_test

import (
	"testing"

	"github.com/aisc/pkg/statestore/storeadapter"
	"github.com/aisc/pkg/statestore/test"
	"github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/storage/inmemstore"
	"github.com/aisc/pkg/storage/leveldbstore"
)

func TestStateStoreAdapter(t *testing.T) {
	t.Parallel()

	test.Run(t, func(t *testing.T) storage.StateStorer {
		t.Helper()

		store, err := storeadapter.NewStateStorerAdapter(inmemstore.New())
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

		leveldb, err := leveldbstore.New(dir, nil)
		if err != nil {
			t.Fatal(err)
		}

		store, err := storeadapter.NewStateStorerAdapter(leveldb)
		if err != nil {
			t.Fatal(err)
		}

		return store
	})
}
