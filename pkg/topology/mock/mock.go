// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/aisc/pkg/p2p"
	"github.com/aisc/pkg/ aisc"
	"github.com/aisc/pkg/topology"
)

type mock struct {
	peers           [] aisc.Address
	depth           uint8
	closestPeer      aisc.Address
	closestPeerErr  error
	peersErr        error
	addPeersErr     error
	isWithinFunc    func(c  aisc.Address) bool
	marshalJSONFunc func() ([]byte, error)
	mtx             sync.Mutex
	health          map[string]bool
}

var _ topology.Driver = (*mock)(nil)

func WithPeers(peers ... aisc.Address) Option {
	return optionFunc(func(d *mock) {
		d.peers = peers
	})
}

func WithAddPeersErr(err error) Option {
	return optionFunc(func(d *mock) {
		d.addPeersErr = err
	})
}

func WithNeighborhoodDepth(dd uint8) Option {
	return optionFunc(func(d *mock) {
		d.depth = dd
	})
}

func WithClosestPeer(addr  aisc.Address) Option {
	return optionFunc(func(d *mock) {
		d.closestPeer = addr
	})
}

func WithClosestPeerErr(err error) Option {
	return optionFunc(func(d *mock) {
		d.closestPeerErr = err
	})
}

func WithMarshalJSONFunc(f func() ([]byte, error)) Option {
	return optionFunc(func(d *mock) {
		d.marshalJSONFunc = f
	})
}

func WithIsWithinFunc(f func( aisc.Address) bool) Option {
	return optionFunc(func(d *mock) {
		d.isWithinFunc = f
	})
}

func NewTopologyDriver(opts ...Option) *mock {
	d := new(mock)
	for _, o := range opts {
		o.apply(d)
	}

	d.health = map[string]bool{}

	return d
}

func (d *mock) AddPeers(addrs ... aisc.Address) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.peers = append(d.peers, addrs...)
}

func (d *mock) Connected(ctx context.Context, peer p2p.Peer, _ bool) error {
	d.AddPeers(peer.Address)
	return nil
}

func (d *mock) Disconnected(peer p2p.Peer) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.peers =  aisc.RemoveAddress(d.peers, peer.Address)
}

func (d *mock) Announce(_ context.Context, _  aisc.Address, _ bool) error {
	return nil
}

func (d *mock) AnnounceTo(_ context.Context, _, _  aisc.Address, _ bool) error {
	return nil
}

func (d *mock) UpdatePeerHealth(peer  aisc.Address, health bool, pingDur time.Duration) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.health[peer.ByteString()] = health
}

func (d *mock) PeersHealth() map[string]bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return maps.Clone(d.health)
}

func (d *mock) Peers() [] aisc.Address {
	return d.peers
}

func (d *mock) ClosestPeer(addr  aisc.Address, wantSelf bool, _ topology.Select, skipPeers ... aisc.Address) (peerAddr  aisc.Address, err error) {
	if len(skipPeers) == 0 {
		if d.closestPeerErr != nil {
			return d.closestPeer, d.closestPeerErr
		}
		if !d.closestPeer.Equal( aisc.ZeroAddress) {
			return d.closestPeer, nil
		}
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()

	if len(d.peers) == 0 {
		return peerAddr, topology.ErrNotFound
	}

	skipPeer := false
	for _, p := range d.peers {
		for _, a := range skipPeers {
			if a.Equal(p) {
				skipPeer = true
				break
			}
		}
		if skipPeer {
			skipPeer = false
			continue
		}

		if peerAddr.IsZero() {
			peerAddr = p
		}

		if closer, _ := p.Closer(addr, peerAddr); closer {
			peerAddr = p
		}
	}

	if peerAddr.IsZero() {
		if wantSelf {
			return peerAddr, topology.ErrWantSelf
		} else {
			return peerAddr, topology.ErrNotFound
		}
	}

	return peerAddr, nil
}

func (m *mock) IsReachable() bool {
	return true
}

func (d *mock) SubscribeTopologyChange() (c <-chan struct{}, unsubscribe func()) {
	return c, unsubscribe
}

func (m *mock) NeighborhoodDepth() uint8 {
	return m.depth
}

func (m *mock) SetStorageRadius(uint8) {}

// EachConnectedPeer implements topology.PeerIterator interface.
func (d *mock) EachConnectedPeer(f topology.EachPeerFunc, _ topology.Select) (err error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.peersErr != nil {
		return d.peersErr
	}

	for i, p := range d.peers {
		_, _, err = f(p, uint8(i))
		if err != nil {
			return
		}
	}

	return nil
}

// EachConnectedPeerRev implements topology.PeerIterator interface.
func (d *mock) EachConnectedPeerRev(f topology.EachPeerFunc, _ topology.Select) (err error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for i := len(d.peers) - 1; i >= 0; i-- {
		_, _, err = f(d.peers[i], uint8(i))
		if err != nil {
			return
		}
	}

	return nil
}

func (d *mock) Snapshot() *topology.KadParams {
	return new(topology.KadParams)
}

func (d *mock) Halt()        {}
func (d *mock) Close() error { return nil }

type Option interface {
	apply(*mock)
}

type optionFunc func(*mock)

func (f optionFunc) apply(r *mock) { f(r) }
