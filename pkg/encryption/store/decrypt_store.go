// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"context"
	"encoding/binary"

	"github.com/aisc/pkg/encryption"
	"github.com/aisc/pkg/file"
	"github.com/aisc/pkg/file/redundancy"
	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
	"golang.org/x/crypto/sha3"
)

type decryptingStore struct {
	storage.Getter
}

func New(s storage.Getter) storage.Getter {
	return &decryptingStore{s}
}

func (s *decryptingStore) Get(ctx context.Context, addr  aisc.Address) (ch  aisc.Chunk, err error) {
	switch l := len(addr.Bytes()); l {
	case  aisc.HashSize:
		// normal, unencrypted content
		return s.Getter.Get(ctx, addr)

	case encryption.ReferenceSize:
		// encrypted reference
		ref := addr.Bytes()
		address :=  aisc.NewAddress(ref[: aisc.HashSize])
		ch, err := s.Getter.Get(ctx, address)
		if err != nil {
			return nil, err
		}

		d, err := DecryptChunkData(ch.Data(), ref[ aisc.HashSize:])
		if err != nil {
			return nil, err
		}
		return  aisc.NewChunk(address, d), nil

	default:
		return nil, storage.ErrReferenceLength
	}
}

func DecryptChunkData(chunkData []byte, encryptionKey encryption.Key) ([]byte, error) {
	decryptedSpan, decryptedData, err := decrypt(chunkData, encryptionKey)
	if err != nil {
		return nil, err
	}

	// removing extra bytes which were just added for padding
	level, span := redundancy.DecodeSpan(decryptedSpan)
	length := binary.LittleEndian.Uint64(span)
	if length >  aisc.ChunkSize {
		dataRefSize := uint64( aisc.HashSize + encryption.KeyLength)
		dataShards, parities := file.ReferenceCount(length, level, true)
		length = dataRefSize*uint64(dataShards) + uint64(parities* aisc.HashSize)
	}

	c := make([]byte, length+8)
	copy(c[:8], decryptedSpan)
	copy(c[8:], decryptedData[:length])

	return c, nil
}

func decrypt(chunkData []byte, key encryption.Key) ([]byte, []byte, error) {
	decryptedSpan, err := newSpanEncryption(key).Decrypt(chunkData[: aisc.SpanSize])
	if err != nil {
		return nil, nil, err
	}
	decryptedData, err := newDataEncryption(key).Decrypt(chunkData[ aisc.SpanSize:])
	if err != nil {
		return nil, nil, err
	}
	return decryptedSpan, decryptedData, nil
}

func newSpanEncryption(key encryption.Key) encryption.Interface {
	refSize := int64( aisc.HashSize + encryption.KeyLength)
	return encryption.New(key, 0, uint32( aisc.ChunkSize/refSize), sha3.NewLegacyKeccak256)
}

func newDataEncryption(key encryption.Key) encryption.Interface {
	return encryption.New(key, int( aisc.ChunkSize), 0, sha3.NewLegacyKeccak256)
}
