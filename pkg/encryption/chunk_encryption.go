// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package encryption

import (
	"github.com/aisc/pkg/ aisc"
	"golang.org/x/crypto/sha3"
)

// ChunkEncrypter encrypts chunk data.
type ChunkEncrypter interface {
	EncryptChunk([]byte) (key Key, encryptedSpan, encryptedData []byte, err error)
}

type chunkEncrypter struct{}

func NewChunkEncrypter() ChunkEncrypter { return &chunkEncrypter{} }

func (c *chunkEncrypter) EncryptChunk(chunkData []byte) (Key, []byte, []byte, error) {
	key := GenerateRandomKey(KeyLength)
	encryptedSpan, err := newSpanEncryption(key).Encrypt(chunkData[:8])
	if err != nil {
		return nil, nil, nil, err
	}
	encryptedData, err := newDataEncryption(key).Encrypt(chunkData[8:])
	if err != nil {
		return nil, nil, nil, err
	}
	return key, encryptedSpan, encryptedData, nil
}

func newSpanEncryption(key Key) Interface {
	refSize := int64( aisc.HashSize + KeyLength)
	return New(key, 0, uint32( aisc.ChunkSize/refSize), sha3.NewLegacyKeccak256)
}

func newDataEncryption(key Key) Interface {
	return New(key, int( aisc.ChunkSize), 0, sha3.NewLegacyKeccak256)
}
