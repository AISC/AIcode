// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cidv1

import (
	"fmt"

	"github.com/aisc/pkg/resolver"
	"github.com/aisc/pkg/ aisc"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// https://github.com/multiformats/multicodec/blob/master/table.csv
const (
	AiscNsCodec       uint64 = 0xe4
	AiscManifestCodec uint64 = 0xfa
	AiscFeedCodec     uint64 = 0xfb
)

type Resolver struct{}

func (Resolver) Resolve(name string) ( aisc.Address, error) {
	id, err := cid.Parse(name)
	if err != nil {
		return  aisc.ZeroAddress, fmt.Errorf("failed parsing CID %s err %w: %w", name, err, resolver.ErrParse)
	}

	switch id.Prefix().GetCodec() {
	case AiscNsCodec:
	case AiscManifestCodec:
	case AiscFeedCodec:
	default:
		return  aisc.ZeroAddress, fmt.Errorf("unsupported codec for CID %d: %w", id.Prefix().GetCodec(), resolver.ErrParse)
	}

	dh, err := multihash.Decode(id.Hash())
	if err != nil {
		return  aisc.ZeroAddress, fmt.Errorf("unable to decode hash %w: %w", err, resolver.ErrInvalidContentHash)
	}

	addr :=  aisc.NewAddress(dh.Digest)

	return addr, nil
}

func (Resolver) Close() error {
	return nil
}
