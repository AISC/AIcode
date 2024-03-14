// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package retrieval

import (
	"context"

	"github.com/aisc/pkg/p2p"
	"github.com/aisc/pkg/ aisc"
)

func (s *Service) Handler(ctx context.Context, p p2p.Peer, stream p2p.Stream) error {
	return s.handler(ctx, p, stream)
}

func (s *Service) ClosestPeer(addr  aisc.Address, skipPeers [] aisc.Address, allowUpstream bool) ( aisc.Address, error) {
	return s.closestPeer(addr, skipPeers, allowUpstream)
}
