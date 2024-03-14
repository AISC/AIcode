// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package inmemchunkstore_test

import (
	"testing"

	"github.com/aisc/pkg/storage/inmemchunkstore"
	"github.com/aisc/pkg/storage/storagetest"
)

func TestTxChunkStore(t *testing.T) {
	t.Parallel()

	storagetest.TestTxChunkStore(t, inmemchunkstore.NewTxChunkStore(inmemchunkstore.New()))
}
