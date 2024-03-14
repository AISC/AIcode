// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p_test

import (
	"testing"

	"github.com/aisc/pkg/p2p"
	"github.com/libp2p/go-libp2p/core/network"
)

func TestNewAiscStreamName(t *testing.T) {
	t.Parallel()

	want := "/ aisc/hive/1.2.0/peers"
	got := p2p.NewAiscStreamName("hive", "1.2.0", "peers")

	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestReachabilityStatus_String(t *testing.T) {
	t.Parallel()

	mapping := map[string]string{
		p2p.ReachabilityStatusUnknown.String(): network.ReachabilityUnknown.String(),
		p2p.ReachabilityStatusPrivate.String(): network.ReachabilityPrivate.String(),
		p2p.ReachabilityStatusPublic.String():  network.ReachabilityPublic.String(),
	}
	for have, want := range mapping {
		if have != want {
			t.Fatalf("have reachability status string %q; want %q", have, want)
		}
	}
}
