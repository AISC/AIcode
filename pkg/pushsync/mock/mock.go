// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"context"

	"github.com/aisc/pkg/pushsync"
	"github.com/aisc/pkg/ aisc"
)

type mock struct {
	sendChunk func(ctx context.Context, chunk  aisc.Chunk) (*pushsync.Receipt, error)
}

func New(sendChunk func(ctx context.Context, chunk  aisc.Chunk) (*pushsync.Receipt, error)) pushsync.PushSyncer {
	return &mock{sendChunk: sendChunk}
}

func (s *mock) PushChunkToClosest(ctx context.Context, chunk  aisc.Chunk) (*pushsync.Receipt, error) {
	return s.sendChunk(ctx, chunk)
}

func (s *mock) Close() error {
	return nil
}
