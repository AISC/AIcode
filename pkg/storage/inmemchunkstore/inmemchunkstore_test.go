// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package inmemchunkstore_test

import (
	"testing"

	inmem "github.com/aisc/pkg/storage/inmemchunkstore"
	"github.com/aisc/pkg/storage/storagetest"
)

func TestChunkStore(t *testing.T) {
	t.Parallel()

	storagetest.TestChunkStore(t, inmem.New())
}

func BenchmarkChunkStore(t *testing.B) {
	storagetest.RunChunkStoreBenchmarkTests(t, inmem.New())
}
