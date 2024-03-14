// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"github.com/aisc/pkg/ aisc"
)

type MockPricer struct {
	peerPrice uint64
	price     uint64
}

func NewMockService(price, peerPrice uint64) *MockPricer {
	return &MockPricer{
		peerPrice: peerPrice,
		price:     price,
	}
}

func (pricer *MockPricer) PeerPrice(peer, chunk  aisc.Address) uint64 {
	return pricer.peerPrice
}

func (pricer *MockPricer) Price(chunk  aisc.Address) uint64 {
	return pricer.price
}
