// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"github.com/aisc/pkg/postage"
	"github.com/aisc/pkg/ aisc"
)

type mockStamper struct{}

// NewStamper returns anew new mock stamper.
func NewStamper() postage.Stamper {
	return &mockStamper{}
}

// Stamp implements the Stamper interface. It returns an empty postage stamp.
func (mockStamper) Stamp(_  aisc.Address) (*postage.Stamp, error) {
	return &postage.Stamp{}, nil
}
