// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package addresses_test

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/aisc/pkg/file"
	"github.com/aisc/pkg/file/addresses"
	"github.com/aisc/pkg/file/joiner"
	filetest "github.com/aisc/pkg/file/testing"
	"github.com/aisc/pkg/storage/inmemchunkstore"
	"github.com/aisc/pkg/ aisc"
)

func TestAddressesGetterIterateChunkAddresses(t *testing.T) {
	t.Parallel()

	store := inmemchunkstore.New()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// create root chunk with 2 references and the referenced data chunks
	rootChunk := filetest.GenerateTestRandomFileChunk( aisc.ZeroAddress,  aisc.ChunkSize*2,  aisc.SectionSize*2)
	err := store.Put(ctx, rootChunk)
	if err != nil {
		t.Fatal(err)
	}

	firstAddress :=  aisc.NewAddress(rootChunk.Data()[8 :  aisc.SectionSize+8])
	firstChunk := filetest.GenerateTestRandomFileChunk(firstAddress,  aisc.ChunkSize,  aisc.ChunkSize)
	err = store.Put(ctx, firstChunk)
	if err != nil {
		t.Fatal(err)
	}

	secondAddress :=  aisc.NewAddress(rootChunk.Data()[ aisc.SectionSize+8:])
	secondChunk := filetest.GenerateTestRandomFileChunk(secondAddress,  aisc.ChunkSize,  aisc.ChunkSize)
	err = store.Put(ctx, secondChunk)
	if err != nil {
		t.Fatal(err)
	}

	createdAddresses := [] aisc.Address{rootChunk.Address(), firstAddress, secondAddress}

	foundAddresses := make(map[string]struct{})
	var foundAddressesMu sync.Mutex

	addressIterFunc := func(addr  aisc.Address) error {
		foundAddressesMu.Lock()
		defer foundAddressesMu.Unlock()

		foundAddresses[addr.String()] = struct{}{}
		return nil
	}

	addressesGetter := addresses.NewGetter(store, addressIterFunc)

	j, _, err := joiner.New(ctx, addressesGetter, store, rootChunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	_, err = file.JoinReadAll(ctx, j, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if len(createdAddresses) != len(foundAddresses) {
		t.Fatalf("expected to find %d addresses, got %d", len(createdAddresses), len(foundAddresses))
	}

	checkAddressFound := func(t *testing.T, foundAddresses map[string]struct{}, address  aisc.Address) {
		t.Helper()

		if _, ok := foundAddresses[address.String()]; !ok {
			t.Fatalf("expected address %s not found", address.String())
		}
	}

	for _, createdAddress := range createdAddresses {
		checkAddressFound(t, foundAddresses, createdAddress)
	}
}
