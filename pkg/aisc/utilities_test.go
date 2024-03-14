// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aisc_test

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/aisc/pkg/aisc"
	"github.com/aisc/pkg/ aisc"
	"github.com/aisc/pkg/util/testutil"
)

func Test_ContainsAddress(t *testing.T) {
	t.Parallel()

	addrs := makeAddreses(t, 10)
	tt := []struct {
		addresses []aisc.Address
		search    aisc.Address
		contains  bool
	}{
		{addresses: nil, search: aisc.Address{}},
		{addresses: nil, search: makeAddress(t)},
		{addresses: make([]aisc.Address, 10), search: aisc.Address{}, contains: true},
		{addresses: makeAddreses(t, 0), search: makeAddress(t)},
		{addresses: makeAddreses(t, 10), search: makeAddress(t)},
		{addresses: addrs, search: addrs[0], contains: true},
		{addresses: addrs, search: addrs[1], contains: true},
		{addresses: addrs, search: addrs[3], contains: true},
		{addresses: addrs, search: addrs[9], contains: true},
	}

	for _, tc := range tt {
		contains := aisc.ContainsAddress(tc.addresses, &tc.search)
		if contains != tc.contains {
			t.Fatalf("got %v, want %v", contains, tc.contains)
		}
	}
}
func makeAddreses(t *testing.T, count int) []aisc.Address {
	t.Helper()

	result := make([]aisc.Address, count)
	for i := 0; i < count; i++ {
		result[i] = makeAddress(t)
	}
	return result
}

func makeAddress(t *testing.T) aisc.Address {
	t.Helper()

	multiaddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/1634/p2p/16Uiu2HAkx8ULY8cTXhdVAcMmLcH9AsTKz6uBQ7DPLKRjMLgBVYkA")
	if err != nil {
		t.Fatal(err)
	}

	return aisc.Address{
		Underlay:        multiaddr,
		Overlay:          aisc.RandAddress(t),
		Signature:       testutil.RandBytes(t, 12),
		Nonce:           testutil.RandBytes(t, 12),
		EthereumAddress: testutil.RandBytes(t, 32),
	}
}
