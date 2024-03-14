// Copyright 2021 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"testing"

	"github.com/aisc/pkg/api"
)

func TestAiscNodeMode_String(t *testing.T) {
	t.Parallel()

	mapping := map[string]string{
		api.UnknownMode.String(): "unknown",
		api.LightMode.String():   "light",
		api.FullMode.String():    "full",
		api.DevMode.String():     "dev",
	}

	for have, want := range mapping {
		if have != want {
			t.Fatalf("unexpected aisc node mode: have %q; want %q", have, want)
		}
	}
}
