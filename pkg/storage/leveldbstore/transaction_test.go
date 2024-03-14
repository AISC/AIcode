// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package leveldbstore_test

import (
	"testing"

	"github.com/aisc/pkg/storage/leveldbstore"
	"github.com/aisc/pkg/storage/storagetest"
)

func TestTxStore(t *testing.T) {
	t.Parallel()

	store, err := leveldbstore.New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	storagetest.TestTxStore(t, leveldbstore.NewTxStore(store))
}
