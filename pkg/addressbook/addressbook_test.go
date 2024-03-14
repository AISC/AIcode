// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package addressbook_test

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/aisc/pkg/addressbook"
	"github.com/aisc/pkg/aisc"
	"github.com/aisc/pkg/crypto"
	"github.com/aisc/pkg/statestore/mock"
	"github.com/aisc/pkg/ aisc"

	ma "github.com/multiformats/go-multiaddr"
)

type bookFunc func() (book addressbook.Interface)

func TestInMem(t *testing.T) {
	t.Parallel()

	run(t, func() addressbook.Interface {
		store := mock.NewStateStore()
		book := addressbook.New(store)
		return book
	})
}

func run(t *testing.T, f bookFunc) {
	t.Helper()

	store := f()
	addr1 :=  aisc.NewAddress([]byte{0, 1, 2, 3})
	addr2 :=  aisc.NewAddress([]byte{0, 1, 2, 4})
	trxHash := common.HexToHash("0x1").Bytes()
	multiaddr, err := ma.NewMultiaddr("/ip4/1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}

	pk, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		t.Fatal(err)
	}

	aiscAddr, err := aisc.NewAddress(crypto.NewDefaultSigner(pk), multiaddr, addr1, 1, trxHash)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Put(addr1, *aiscAddr)
	if err != nil {
		t.Fatal(err)
	}

	v, err := store.Get(addr1)
	if err != nil {
		t.Fatal(err)
	}

	if !aiscAddr.Equal(v) {
		t.Fatalf("expectted: %s, want %s", v, multiaddr)
	}

	notFound, err := store.Get(addr2)
	if !errors.Is(err, addressbook.ErrNotFound) {
		t.Fatal(err)
	}

	if notFound != nil {
		t.Fatalf("expected nil got %s", v)
	}

	overlays, err := store.Overlays()
	if err != nil {
		t.Fatal(err)
	}

	if len(overlays) != 1 {
		t.Fatalf("expected overlay len %v, got %v", 1, len(overlays))
	}

	addresses, err := store.Addresses()
	if err != nil {
		t.Fatal(err)
	}

	if len(addresses) != 1 {
		t.Fatalf("expected addresses len %v, got %v", 1, len(addresses))
	}
}
