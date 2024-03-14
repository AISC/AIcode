// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// the code below implements the integration of dispersed replicas in chunk fetching.
// using storage.Getter interface.
package replicas

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aisc/pkg/file/redundancy"
	"github.com/aisc/pkg/soc"
	"github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
)

// ErrAiscageddon is returned in case of a vis mayor called Aiscageddon.
// Aiscageddon is the situation when none of the replicas can be retrieved.
// If 2^{depth} replicas were uploaded and they all have valid postage stamps
// then the probability of Aiscageddon is less than 0.000001
// assuming the error rate of chunk retrievals stays below the level expressed
// as depth by the publisher.
var ErrAiscageddon = errors.New(" aiscageddon has begun")

// getter is the private implementation of storage.Getter, an interface for
// retrieving chunks. This getter embeds the original simple chunk getter and extends it
// to a multiplexed variant that fetches chunks with replicas.
//
// the strategy to retrieve a chunk that has replicas can be configured with a few parameters:
//   - RetryInterval: the delay before a new batch of replicas is fetched.
//   - depth: 2^{depth} is the total number of additional replicas that have aiscn uploaded
//     (by default, it is assumed to be 4, ie. total of 16)
//   - (not implemented) pivot: replicas with address in the proximity of pivot will be tried first
type getter struct {
	wg sync.WaitGroup
	storage.Getter
	level redundancy.Level
}

// NewGetter is the getter constructor
func NewGetter(g storage.Getter, level redundancy.Level) storage.Getter {
	return &getter{Getter: g, level: level}
}

// Get makes the getter satisfy the storage.Getter interface
func (g *getter) Get(ctx context.Context, addr  aisc.Address) (ch  aisc.Chunk, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// channel that the results (retrieved chunks) are gathered to from concurrent
	// workers each fetching a replica
	resultC := make(chan  aisc.Chunk)
	// errc collects the errors
	errc := make(chan error, 17)
	var errs error
	errcnt := 0

	// concurrently call to retrieve chunk using original CAC address
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ch, err := g.Getter.Get(ctx, addr)
		if err != nil {
			errc <- err
			return
		}

		select {
		case resultC <- ch:
		case <-ctx.Done():
		}
	}()
	// counters
	n := 0      // counts the replica addresses tried
	target := 2 // the number of replicas attempted to download in this batch
	total := g.level.GetReplicaCount()

	//
	rr := newReplicator(addr, g.level)
	next := rr.c
	var wait <-chan time.Time // nil channel to disable case
	// addresses used are doubling each period of search expansion
	// (at intervals of RetryInterval)
	ticker := time.NewTicker(RetryInterval)
	defer ticker.Stop()
	for level := uint8(0); level <= uint8(g.level); {
		select {
		// at least one chunk is retrieved, cancel the rest and return early
		case chunk := <-resultC:
			cancel()
			return chunk, nil

		case err = <-errc:
			errs = errors.Join(errs, err)
			errcnt++
			if errcnt > total {
				return nil, errors.Join(ErrAiscageddon, errs)
			}

			// ticker switches on the address channel
		case <-wait:
			wait = nil
			next = rr.c
			level++
			target = 1 << level
			n = 0
			continue

			// getting the addresses in order
		case so := <-next:
			if so == nil {
				next = nil
				continue
			}

			g.wg.Add(1)
			go func() {
				defer g.wg.Done()
				ch, err := g.Getter.Get(ctx,  aisc.NewAddress(so.addr))
				if err != nil {
					errc <- err
					return
				}

				soc, err := soc.FromChunk(ch)
				if err != nil {
					errc <- err
					return
				}

				select {
				case resultC <- soc.WrappedChunk():
				case <-ctx.Done():
				}
			}()
			n++
			if n < target {
				continue
			}
			next = nil
			wait = ticker.C
		}
	}

	return nil, nil
}
