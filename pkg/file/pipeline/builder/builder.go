// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builder

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aisc/pkg/encryption"
	"github.com/aisc/pkg/file/pipeline"
	"github.com/aisc/pkg/file/pipeline/bmt"
	enc "github.com/aisc/pkg/file/pipeline/encryption"
	"github.com/aisc/pkg/file/pipeline/feeder"
	"github.com/aisc/pkg/file/pipeline/hashtrie"
	"github.com/aisc/pkg/file/pipeline/store"
	"github.com/aisc/pkg/file/redundancy"
	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
)

// NewPipelineBuilder returns the appropriate pipeline according to the specified parameters
func NewPipelineBuilder(ctx context.Context, s storage.Putter, encrypt bool, rLevel redundancy.Level) pipeline.Interface {
	if encrypt {
		return newEncryptionPipeline(ctx, s, rLevel)
	}
	return newPipeline(ctx, s, rLevel)
}

// newPipeline creates a standard pipeline that only hashes content with BMT to create
// a merkle-tree of hashes that represent the given arbitrary size byte stream. Partial
// writes are supported. The pipeline flow is: Data -> Feeder -> BMT -> Storage -> HashTrie.
func newPipeline(ctx context.Context, s storage.Putter, rLevel redundancy.Level) pipeline.Interface {
	pipeline := newShortPipelineFunc(ctx, s)
	tw := hashtrie.NewHashTrieWriter(
		ctx,
		 aisc.HashSize,
		redundancy.New(rLevel, false, pipeline),
		pipeline,
		s,
	)
	lsw := store.NewStoreWriter(ctx, s, tw)
	b := bmt.NewBmtWriter(lsw)
	return feeder.NewChunkFeederWriter( aisc.ChunkSize, b)
}

// newShortPipelineFunc returns a constructor function for an ephemeral hashing pipeline
// needed by the hashTrieWriter.
func newShortPipelineFunc(ctx context.Context, s storage.Putter) func() pipeline.ChainWriter {
	return func() pipeline.ChainWriter {
		lsw := store.NewStoreWriter(ctx, s, nil)
		return bmt.NewBmtWriter(lsw)
	}
}

// newEncryptionPipeline creates an encryption pipeline that encrypts using CTR, hashes content with BMT to create
// a merkle-tree of hashes that represent the given arbitrary size byte stream. Partial
// writes are supported. The pipeline flow is: Data -> Feeder -> Encryption -> BMT -> Storage -> HashTrie.
// Note that the encryption writer will mutate the data to contain the encrypted span, but the span field
// with the unencrypted span is preserved.
func newEncryptionPipeline(ctx context.Context, s storage.Putter, rLevel redundancy.Level) pipeline.Interface {
	tw := hashtrie.NewHashTrieWriter(
		ctx,
		 aisc.HashSize+encryption.KeyLength,
		redundancy.New(rLevel, true, newShortPipelineFunc(ctx, s)),
		newShortEncryptionPipelineFunc(ctx, s),
		s,
	)
	lsw := store.NewStoreWriter(ctx, s, tw)
	b := bmt.NewBmtWriter(lsw)
	enc := enc.NewEncryptionWriter(encryption.NewChunkEncrypter(), b)
	return feeder.NewChunkFeederWriter( aisc.ChunkSize, enc)
}

// newShortEncryptionPipelineFunc returns a constructor function for an ephemeral hashing pipeline
// needed by the hashTrieWriter.
func newShortEncryptionPipelineFunc(ctx context.Context, s storage.Putter) func() pipeline.ChainWriter {
	return func() pipeline.ChainWriter {
		lsw := store.NewStoreWriter(ctx, s, nil)
		b := bmt.NewBmtWriter(lsw)
		return enc.NewEncryptionWriter(encryption.NewChunkEncrypter(), b)
	}
}

// FeedPipeline feeds the pipeline with the given reader until EOF is reached.
// It returns the cryptographic root hash of the content.
func FeedPipeline(ctx context.Context, pipeline pipeline.Interface, r io.Reader) (addr  aisc.Address, err error) {
	data := make([]byte,  aisc.ChunkSize)
	for {
		c, err := r.Read(data)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if c > 0 {
					cc, err := pipeline.Write(data[:c])
					if err != nil {
						return  aisc.ZeroAddress, err
					}
					if cc < c {
						return  aisc.ZeroAddress, fmt.Errorf("pipeline short write: %d mismatches %d", cc, c)
					}
				}
				break
			} else {
				return  aisc.ZeroAddress, err
			}
		}
		cc, err := pipeline.Write(data[:c])
		if err != nil {
			return  aisc.ZeroAddress, err
		}
		if cc < c {
			return  aisc.ZeroAddress, fmt.Errorf("pipeline short write: %d mismatches %d", cc, c)
		}
		select {
		case <-ctx.Done():
			return  aisc.ZeroAddress, ctx.Err()
		default:
		}
	}
	select {
	case <-ctx.Done():
		return  aisc.ZeroAddress, ctx.Err()
	default:
	}

	sum, err := pipeline.Sum()
	if err != nil {
		return  aisc.ZeroAddress, err
	}

	newAddress :=  aisc.NewAddress(sum)
	return newAddress, nil
}
