// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stampindex

import "github.com/aisc/pkg/ aisc"

var (
	ErrStampItemMarshalNamespaceInvalid  = errStampItemMarshalNamespaceInvalid
	ErrStampItemMarshalBatchIndexInvalid = errStampItemMarshalBatchIndexInvalid
	ErrStampItemMarshalBatchIDInvalid    = errStampItemMarshalBatchIDInvalid
	ErrStampItemUnmarshalInvalidSize     = errStampItemUnmarshalInvalidSize
)

// NewItemWithValues creates a new Item with given values and fixed keys.
func NewItemWithValues(batchTimestamp []byte, chunkAddress  aisc.Address, chunkIsImmutable bool) *Item {
	return &Item{
		namespace:  []byte("test_namespace"),
		batchID:    []byte{ aisc.HashSize - 1: 9},
		stampIndex: []byte{ aisc.StampIndexSize - 1: 9},

		StampTimestamp:   batchTimestamp,
		ChunkAddress:     chunkAddress,
		ChunkIsImmutable: chunkIsImmutable,
	}
}

// NewItemWithKeys creates a new Item with given keys and zero values.
func NewItemWithKeys(namespace string, batchID, batchIndex []byte) *Item {
	return &Item{
		namespace:  append([]byte(nil), namespace...),
		batchID:    batchID,
		stampIndex: batchIndex,
	}
}
