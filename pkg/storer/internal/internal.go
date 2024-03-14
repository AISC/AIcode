// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal

import (
	"bytes"
	"context"
	"errors"

	"github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/storage/inmemchunkstore"
	"github.com/aisc/pkg/storage/inmemstore"
	"github.com/aisc/pkg/ aisc"
)

// Storage groups the storage.Store and storage.ChunkStore interfaces.
type Storage interface {
	IndexStore() storage.BatchedStore
	ChunkStore() storage.ChunkStore
}

// PutterCloserWithReference provides a Putter which can be closed with a root
//  aisc reference associated with this session.
type PutterCloserWithReference interface {
	Put(context.Context, Storage, storage.Writer,  aisc.Chunk) error
	Close(Storage, storage.Writer,  aisc.Address) error
	Cleanup(TxExecutor) error
}

// TxExecutor executes a function in a transaction.
type TxExecutor interface {
	Execute(context.Context, func(Storage) error) error
}

var emptyAddr = make([]byte,  aisc.HashSize)

// AddressOrZero returns  aisc.ZeroAddress if the buf is of zero bytes. The Zero byte
// buffer is used by the items to serialize their contents and if valid  aisc.ZeroAddress
// entries are allowed.
func AddressOrZero(buf []byte)  aisc.Address {
	if bytes.Equal(buf, emptyAddr) {
		return  aisc.ZeroAddress
	}
	return  aisc.NewAddress(append(make([]byte, 0,  aisc.HashSize), buf...))
}

// AddressBytesOrZero is a helper which creates a zero buffer of  aisc.HashSize. This
// is required during storing the items in the Store as their serialization formats
// are strict.
func AddressBytesOrZero(addr  aisc.Address) []byte {
	if addr.IsZero() {
		return make([]byte,  aisc.HashSize)
	}
	return addr.Bytes()
}

// BatchedStorage groups the Storage and TxExecutor interfaces.
type BatchedStorage interface {
	Storage
	TxExecutor
}

// NewInmemStorage constructs a inmem Storage implementation which can be used
// for the tests in the internal packages.
func NewInmemStorage() (BatchedStorage, func() error) {
	ts := &inmemRepository{
		indexStore: inmemstore.New(),
		chunkStore: inmemchunkstore.New(),
	}

	return ts, func() error {
		return errors.Join(ts.indexStore.Close(), ts.chunkStore.Close())
	}
}

type inmemRepository struct {
	indexStore storage.BatchedStore
	chunkStore storage.ChunkStore
}

func (t *inmemRepository) IndexStore() storage.BatchedStore                       { return t.indexStore }
func (t *inmemRepository) ChunkStore() storage.ChunkStore                         { return t.chunkStore }
func (t *inmemRepository) Execute(_ context.Context, f func(Storage) error) error { return f(t) }
