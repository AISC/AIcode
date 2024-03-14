// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libp2p_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/aisc/pkg/addressbook"
	"github.com/aisc/pkg/crypto"
	"github.com/aisc/pkg/log"
	"github.com/aisc/pkg/p2p"
	"github.com/aisc/pkg/p2p/libp2p"
	"github.com/aisc/pkg/spinlock"
	"github.com/aisc/pkg/statestore/mock"
	"github.com/aisc/pkg/ aisc"
	"github.com/aisc/pkg/topology/lightnode"
	"github.com/aisc/pkg/util/testutil"
	"github.com/multiformats/go-multiaddr"
)

type libp2pServiceOpts struct {
	Logger      log.Logger
	Addressbook addressbook.Interface
	PrivateKey  *ecdsa.PrivateKey
	MockPeerKey *ecdsa.PrivateKey
	libp2pOpts  libp2p.Options
	lightNodes  *lightnode.Container
	notifier    p2p.PickyNotifier
}

// newService constructs a new libp2p service.
func newService(t *testing.T, networkID uint64, o libp2pServiceOpts) (s *libp2p.Service, overlay  aisc.Address) {
	t.Helper()

	 aiscKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		t.Fatal(err)
	}

	nonce := common.HexToHash("0x1").Bytes()

	overlay, err = crypto.NewOverlayAddress( aiscKey.PublicKey, networkID, nonce)
	if err != nil {
		t.Fatal(err)
	}

	addr := ":0"

	if o.Logger == nil {
		o.Logger = log.Noop
	}

	statestore := mock.NewStateStore()
	if o.Addressbook == nil {
		o.Addressbook = addressbook.New(statestore)
	}

	if o.PrivateKey == nil {
		libp2pKey, err := crypto.GenerateSecp256k1Key()
		if err != nil {
			t.Fatal(err)
		}

		o.PrivateKey = libp2pKey
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if o.lightNodes == nil {
		o.lightNodes = lightnode.NewContainer(overlay)
	}
	opts := o.libp2pOpts
	opts.Nonce = nonce

	s, err = libp2p.New(ctx, crypto.NewDefaultSigner( aiscKey), networkID, overlay, addr, o.Addressbook, statestore, o.lightNodes, o.Logger, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	testutil.CleanupCloser(t, s)

	if o.notifier != nil {
		s.SetPickyNotifier(o.notifier)
	}

	_ = s.Ready()

	return s, overlay
}

// expectPeers validates that peers with addresses are connected.
func expectPeers(t *testing.T, s *libp2p.Service, addrs ... aisc.Address) {
	t.Helper()

	peers := s.Peers()

	if len(peers) != len(addrs) {
		t.Fatalf("got peers %v, want %v", len(peers), len(addrs))
	}

	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i].Bytes(), addrs[j].Bytes()) == -1
	})
	sort.Slice(peers, func(i, j int) bool {
		return bytes.Compare(peers[i].Address.Bytes(), peers[j].Address.Bytes()) == -1
	})

	for i, got := range peers {
		want := addrs[i]
		if !got.Address.Equal(want) {
			t.Errorf("got %v peer %s, want %s", i, got.Address, want)
		}
	}
}

// expectPeersEventually validates that peers with addresses are connected with
// retries. It is supposed to be used to validate asynchronous connecting on the
// peer that is connected to.
func expectPeersEventually(t *testing.T, s *libp2p.Service, addrs ... aisc.Address) {
	t.Helper()

	var peers []p2p.Peer
	err := spinlock.Wait(time.Second, func() bool {
		peers = s.Peers()
		return len(peers) == len(addrs)

	})
	if err != nil {
		t.Fatalf("timed out waiting for peers, got  %v, want %v", len(peers), len(addrs))
	}

	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i].Bytes(), addrs[j].Bytes()) == -1
	})
	sort.Slice(peers, func(i, j int) bool {
		return bytes.Compare(peers[i].Address.Bytes(), peers[j].Address.Bytes()) == -1
	})

	for i, got := range peers {
		want := addrs[i]
		if !got.Address.Equal(want) {
			t.Errorf("got %v peer %s, want %s", i, got.Address, want)
		}
	}
}

func serviceUnderlayAddress(t *testing.T, s *libp2p.Service) multiaddr.Multiaddr {
	t.Helper()

	addrs, err := s.Addresses()
	if err != nil {
		t.Fatal(err)
	}
	return addrs[0]
}
