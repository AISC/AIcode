// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"context"
	"sync"

	"github.com/aisc/pkg/ aisc"
)

type Discovery struct {
	mtx           sync.Mutex
	ctr           int //how many ops
	records       map[string][] aisc.Address
	broadcastFunc func(context.Context,  aisc.Address, ... aisc.Address) error
}

type Option interface {
	apply(*Discovery)
}
type optionFunc func(*Discovery)

func (f optionFunc) apply(r *Discovery) { f(r) }

func WithBroadcastPeers(f func(context.Context,  aisc.Address, ... aisc.Address) error) optionFunc {
	return optionFunc(func(r *Discovery) {
		r.broadcastFunc = f
	})
}

func NewDiscovery(opts ...Option) *Discovery {
	d := &Discovery{
		records: make(map[string][] aisc.Address),
	}
	for _, opt := range opts {
		opt.apply(d)
	}
	return d
}

func (d *Discovery) BroadcastPeers(ctx context.Context, addressee  aisc.Address, peers ... aisc.Address) error {
	if d.broadcastFunc != nil {
		return d.broadcastFunc(ctx, addressee, peers...)
	}
	for _, peer := range peers {
		d.mtx.Lock()
		d.records[addressee.String()] = append(d.records[addressee.String()], peer)
		d.mtx.Unlock()
	}
	d.mtx.Lock()
	d.ctr++
	d.mtx.Unlock()
	return nil
}

func (d *Discovery) Broadcasts() int {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return d.ctr
}

func (d *Discovery) AddresseeRecords(addressee  aisc.Address) (peers [] aisc.Address, exists bool) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	peers, exists = d.records[addressee.String()]
	return
}

func (d *Discovery) Reset() {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.ctr = 0
	d.records = make(map[string][] aisc.Address)
}
