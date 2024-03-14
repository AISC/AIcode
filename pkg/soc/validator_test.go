// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package soc_test

import (
	"crypto/rand"
	"io"
	"strings"
	"testing"

	"github.com/aisc/pkg/cac"
	"github.com/aisc/pkg/crypto"
	"github.com/aisc/pkg/soc"
	"github.com/aisc/pkg/ aisc"
)

// TestValid verifies that the validator can detect
// valid soc chunks.
func TestValid(t *testing.T) {
	t.Parallel()

	socAddress :=  aisc.MustParseHexAddress("9d453ebb73b2fedaaf44ceddcf7a0aa37f3e3d6453fea5841c31f0ea6d61dc85")

	// signed soc chunk of:
	// id: 0
	// wrapped chunk of: `foo`
	// owner: 0x8d3766440f0d7b949a5e32995d09619a7f86e632
	sch :=  aisc.NewChunk(socAddress, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 90, 205, 56, 79, 235, 193, 51, 183, 178, 69, 229, 221, 198, 45, 130, 210, 205, 237, 145, 130, 210, 113, 97, 38, 205, 136, 68, 80, 154, 246, 90, 5, 61, 235, 65, 130, 8, 2, 127, 84, 142, 62, 136, 52, 58, 246, 248, 74, 135, 114, 251, 60, 235, 192, 161, 131, 58, 14, 167, 236, 12, 19, 72, 49, 27, 3, 0, 0, 0, 0, 0, 0, 0, 102, 111, 111})

	// check valid chunk
	if !soc.Valid(sch) {
		t.Fatal("valid chunk evaluates to invalid")
	}
}

// TestValidDispersedReplica verifies that the validator can detect
// valid dispersed replicas chunks.
func TestValidDispersedReplica(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		privKey, _ := crypto.DecodeSecp256k1PrivateKey(append([]byte{1}, make([]byte, 31)...))
		signer := crypto.NewDefaultSigner(privKey)

		chData := make([]byte,  aisc.ChunkSize)
		_, _ = io.ReadFull(rand.Reader, chData)
		ch, err := cac.New(chData)
		if err != nil {
			t.Fatal(err)
		}
		id := append([]byte{1}, ch.Address().Bytes()[1:]...)

		socCh, err := soc.New(id, ch).Sign(signer)
		if err != nil {
			t.Fatal(err)
		}

		// check valid chunk
		if !soc.Valid(socCh) {
			t.Fatal("dispersed replica chunk is invalid")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		privKey, _ := crypto.DecodeSecp256k1PrivateKey(append([]byte{1}, make([]byte, 31)...))
		signer := crypto.NewDefaultSigner(privKey)

		chData := make([]byte,  aisc.ChunkSize)
		_, _ = io.ReadFull(rand.Reader, chData)
		ch, err := cac.New(chData)
		if err != nil {
			t.Fatal(err)
		}
		id := append([]byte{1}, ch.Address().Bytes()[1:]...)
		// change to invalid ID
		id[2] += 1

		socCh, err := soc.New(id, ch).Sign(signer)
		if err != nil {
			t.Fatal(err)
		}

		// check valid chunk
		if soc.Valid(socCh) {
			t.Fatal("dispersed replica should be invalid")
		}
	})
}

// TestInvalid verifies that the validator can detect chunks
// with invalid data and invalid address.
func TestInvalid(t *testing.T) {
	t.Parallel()

	socAddress :=  aisc.MustParseHexAddress("9d453ebb73b2fedaaf44ceddcf7a0aa37f3e3d6453fea5841c31f0ea6d61dc85")
	// signed soc chunk of:
	// id: 0
	// wrapped chunk of: `foo`
	// owner: 0x8d3766440f0d7b949a5e32995d09619a7f86e632

	makeSocData := func() []byte {
		return []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 90, 205, 56, 79, 235, 193, 51, 183, 178, 69, 229, 221, 198, 45, 130, 210, 205, 237, 145, 130, 210, 113, 97, 38, 205, 136, 68, 80, 154, 246, 90, 5, 61, 235, 65, 130, 8, 2, 127, 84, 142, 62, 136, 52, 58, 246, 248, 74, 135, 114, 251, 60, 235, 192, 161, 131, 58, 14, 167, 236, 12, 19, 72, 49, 27, 3, 0, 0, 0, 0, 0, 0, 0, 102, 111, 111}
	}

	for _, c := range []struct {
		name  string
		chunk func()  aisc.Chunk
	}{
		{
			name: "wrong soc address",
			chunk: func()  aisc.Chunk {
				wrongAddressBytes := socAddress.Clone().Bytes()
				wrongAddressBytes[0] = 255 - wrongAddressBytes[0]
				wrongAddress :=  aisc.NewAddress(wrongAddressBytes)
				data := makeSocData()
				return  aisc.NewChunk(wrongAddress, data)
			},
		},
		{
			name: "invalid data",
			chunk: func()  aisc.Chunk {
				addr := socAddress.Clone()
				data := makeSocData()
				cursor :=  aisc.HashSize +  aisc.SocSignatureSize
				chunkData := data[cursor:]
				chunkData[0] = 0x01
				return  aisc.NewChunk(addr, data)
			},
		},
		{
			name: "invalid id",
			chunk: func()  aisc.Chunk {
				addr := socAddress.Clone()
				data := makeSocData()
				id := data[: aisc.HashSize]
				id[0] = 0x01
				return  aisc.NewChunk(addr, data)
			},
		},
		{
			name: "invalid signature",
			chunk: func()  aisc.Chunk {
				addr := socAddress.Clone()
				data := makeSocData()
				// modify signature
				cursor :=  aisc.HashSize +  aisc.SocSignatureSize
				sig := data[ aisc.HashSize:cursor]
				sig[0] = 0x01
				return  aisc.NewChunk(addr, data)
			},
		},
		{
			name: "nil data",
			chunk: func()  aisc.Chunk {
				addr := socAddress.Clone()
				return  aisc.NewChunk(addr, nil)
			},
		},
		{
			name: "small data",
			chunk: func()  aisc.Chunk {
				addr := socAddress.Clone()
				return  aisc.NewChunk(addr, []byte("small"))
			},
		},
		{
			name: "large data",
			chunk: func()  aisc.Chunk {
				addr := socAddress.Clone()
				return  aisc.NewChunk(addr, []byte(strings.Repeat("a",  aisc.ChunkSize+ aisc.SpanSize+1)))
			},
		},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if soc.Valid(c.chunk()) {
				t.Fatal("chunk with invalid data evaluates to valid")
			}
		})
	}
}
