// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package file

import (
	"bytes"
	"errors"

	"github.com/aisc/pkg/file/redundancy"
	"github.com/aisc/pkg/ aisc"
)

var (
	zeroAddress = [32]byte{}
)

// ChunkPayloadSize returns the effective byte length of an intermediate chunk
// assumes data is always chunk size (without span)
func ChunkPayloadSize(data []byte) (int, error) {
	l := len(data)
	for l >=  aisc.HashSize {
		if !bytes.Equal(data[l- aisc.HashSize:l], zeroAddress[:]) {
			return l, nil
		}

		l -=  aisc.HashSize
	}

	return 0, errors.New("redundancy getter: intermediate chunk does not have at least a child")
}

// ChunkAddresses returns data shards and parities of the intermediate chunk
// assumes data is truncated by ChunkPayloadSize
func ChunkAddresses(data []byte, parities, reflen int) (addrs [] aisc.Address, shardCnt int) {
	shardCnt = (len(data) - parities* aisc.HashSize) / reflen
	for offset := 0; offset < len(data); offset += reflen {
		addrs = append(addrs,  aisc.NewAddress(data[offset:offset+ aisc.HashSize]))
		if len(addrs) == shardCnt && reflen !=  aisc.HashSize {
			reflen =  aisc.HashSize
			offset += reflen
		}
	}
	return addrs, shardCnt
}

// ReferenceCount brute-forces the data shard count from which identify the parity count as well in a substree
// assumes span >  aisc.chunkSize
// returns data and parity shard number
func ReferenceCount(span uint64, level redundancy.Level, encrytedChunk bool) (int, int) {
	// assume we have a trie of size `span` then we can assume that all of
	// the forks except for the last one on the right are of equal size
	// this is due to how the splitter wraps levels.
	// first the algorithm will search for a BMT level where span can be included
	// then identify how large data one reference can hold on that level
	// then count how many references can satisfy span
	// and finally how many parity shards should be on that level
	maxShards := level.GetMaxShards()
	if encrytedChunk {
		maxShards = level.GetMaxEncShards()
	}
	var (
		branching  = uint64(maxShards) // branching factor is how many data shard references can fit into one intermediate chunk
		branchSize = uint64( aisc.ChunkSize)
	)
	// search for branch level big enough to include span
	branchLevel := 1
	for {
		if branchSize >= span {
			break
		}
		branchSize *= branching
		branchLevel++
	}
	// span in one full reference
	referenceSize := uint64( aisc.ChunkSize)
	// referenceSize = branching ** (branchLevel - 1)
	for i := 1; i < branchLevel-1; i++ {
		referenceSize *= branching
	}

	dataShardAddresses := 1
	spanOffset := referenceSize
	for spanOffset < span {
		spanOffset += referenceSize
		dataShardAddresses++
	}

	parityAddresses := level.GetParities(dataShardAddresses)
	if encrytedChunk {
		parityAddresses = level.GetEncParities(dataShardAddresses)
	}

	return dataShardAddresses, parityAddresses
}
