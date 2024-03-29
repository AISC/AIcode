// Copyright 2021 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"context"

	"github.com/aisc/pkg/postage"
	"github.com/aisc/pkg/ aisc"
)

// Steward represents steward.Interface mock.
type Steward struct {
	addr  aisc.Address
}

// Reupload implements steward.Interface Reupload method.
// The given address is recorded.
func (s *Steward) Reupload(_ context.Context, addr  aisc.Address, _ postage.Stamper) error {
	s.addr = addr
	return nil
}

// IsRetrievable implements steward.Interface IsRetrievable method.
// The method always returns true.
func (s *Steward) IsRetrievable(_ context.Context, addr  aisc.Address) (bool, error) {
	return addr.Equal(s.addr), nil
}

// LastAddress returns the last address given to the Reupload method call.
func (s *Steward) LastAddress()  aisc.Address {
	return s.addr
}
