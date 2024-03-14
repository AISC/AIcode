// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package addresses

import (
	"context"

	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
)

type addressesGetterStore struct {
	getter storage.Getter
	fn      aisc.AddressIterFunc
}

// NewGetter creates a new proxy storage.Getter which calls provided function
// for each chunk address processed.
func NewGetter(getter storage.Getter, fn  aisc.AddressIterFunc) storage.Getter {
	return &addressesGetterStore{getter, fn}
}

func (s *addressesGetterStore) Get(ctx context.Context, addr  aisc.Address) ( aisc.Chunk, error) {
	ch, err := s.getter.Get(ctx, addr)
	if err != nil {
		return nil, err
	}

	return ch, s.fn(ch.Address())
}
