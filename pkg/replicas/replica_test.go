// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// replicas_test just contains helper functions to verify dispersion and replication
package replicas_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/aisc/pkg/file/redundancy"
	"github.com/aisc/pkg/soc"
	"github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
)

// dispersed verifies that a set of addresses are maximally dispersed without repetition
func dispersed(level redundancy.Level, ch  aisc.Chunk, addrs [] aisc.Address) error {
	nhoods := make(map[byte]bool)

	for _, addr := range addrs {
		if len(addr.Bytes()) !=  aisc.HashSize {
			return errors.New("corrupt data: invalid address length")
		}
		nh := addr.Bytes()[0] >> (8 - int(level))
		if nhoods[nh] {
			return errors.New("not dispersed enough: duplicate neighbourhood")
		}
		nhoods[nh] = true
	}
	if len(nhoods) != len(addrs) {
		return fmt.Errorf("not dispersed enough: unexpected number of neighbourhood covered: want %v. got %v", len(addrs), len(nhoods))
	}

	return nil
}

// replicated verifies that the replica chunks are indeed replicas
// of the original chunk wrapped in soc
func replicated(store storage.ChunkStore, ch  aisc.Chunk, addrs [] aisc.Address) error {
	ctx := context.Background()
	for _, addr := range addrs {
		chunk, err := store.Get(ctx, addr)
		if err != nil {
			return err
		}

		sch, err := soc.FromChunk(chunk)
		if err != nil {
			return err
		}
		if !sch.WrappedChunk().Equal(ch) {
			return errors.New("invalid replica: does not wrap original content addressed chunk")
		}
	}
	return nil
}
