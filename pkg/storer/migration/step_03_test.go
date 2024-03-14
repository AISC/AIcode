// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migration_test

import (
	"context"
	"errors"
	"testing"

	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/storage/inmemchunkstore"
	"github.com/aisc/pkg/storage/inmemstore"
	chunktest "github.com/aisc/pkg/storage/testing"
	"github.com/aisc/pkg/storer/internal/reserve"
	localmigration "github.com/aisc/pkg/storer/migration"
	"github.com/aisc/pkg/ aisc"
	"github.com/stretchr/testify/assert"
)

func Test_Step_03(t *testing.T) {
	t.Parallel()

	store := inmemstore.New()
	chStore := inmemchunkstore.New()
	baseAddr :=  aisc.RandAddress(t)
	stepFn := localmigration.Step_03(chStore, func(_  aisc.Chunk)  aisc.ChunkType {
		return  aisc.ChunkTypeContentAddressed
	})

	var chunksPO = make([][] aisc.Chunk, 5)
	var chunksPerPO uint64 = 2

	for i := uint8(0); i <  aisc.MaxBins; i++ {
		err := store.Put(&reserve.BinItem{Bin: i, BinID: 10})
		assert.NoError(t, err)
	}

	for b := 0; b < 5; b++ {
		for i := uint64(0); i < chunksPerPO; i++ {
			ch := chunktest.GenerateTestRandomChunkAt(t, baseAddr, b)
			cb := &reserve.ChunkBinItem{
				Bin: uint8(b),
				// Assign 0 binID to all items to see if migration fixes this
				BinID:     0,
				Address:   ch.Address(),
				BatchID:   ch.Stamp().BatchID(),
				ChunkType:  aisc.ChunkTypeContentAddressed,
			}
			err := store.Put(cb)
			if err != nil {
				t.Fatal(err)
			}

			br := &reserve.BatchRadiusItem{
				BatchID: ch.Stamp().BatchID(),
				Bin:     uint8(b),
				Address: ch.Address(),
				BinID:   0,
			}
			err = store.Put(br)
			if err != nil {
				t.Fatal(err)
			}

			if b < 2 {
				// simulate missing chunkstore entries
				continue
			}

			err = chStore.Put(context.Background(), ch)
			if err != nil {
				t.Fatal(err)
			}

			chunksPO[b] = append(chunksPO[b], ch)
		}
	}

	assert.NoError(t, stepFn(store))

	binIDs := make(map[uint8][]uint64)
	cbCount := 0
	err := store.Iterate(
		storage.Query{Factory: func() storage.Item { return &reserve.ChunkBinItem{} }},
		func(res storage.Result) (stop bool, err error) {
			cb := res.Entry.(*reserve.ChunkBinItem)
			if cb.ChunkType !=  aisc.ChunkTypeContentAddressed {
				return false, errors.New("chunk type should be content addressed")
			}
			binIDs[cb.Bin] = append(binIDs[cb.Bin], cb.BinID)
			cbCount++
			return false, nil
		},
	)
	assert.NoError(t, err)

	for b := 0; b < 5; b++ {
		if b < 2 {
			if _, found := binIDs[uint8(b)]; found {
				t.Fatalf("bin %d should not have any binIDs", b)
			}
			continue
		}
		assert.Len(t, binIDs[uint8(b)], 2)
		for idx, binID := range binIDs[uint8(b)] {
			assert.Equal(t, uint64(idx+1), binID)
		}
	}

	brCount := 0
	err = store.Iterate(
		storage.Query{Factory: func() storage.Item { return &reserve.BatchRadiusItem{} }},
		func(res storage.Result) (stop bool, err error) {
			br := res.Entry.(*reserve.BatchRadiusItem)
			if br.Bin < 2 {
				return false, errors.New("bin should be >= 2")
			}
			brCount++
			return false, nil
		},
	)
	assert.NoError(t, err)

	assert.Equal(t, cbCount, brCount)
}
