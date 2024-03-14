// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock_test

import (
	"testing"

	"github.com/aisc/pkg/statestore/mock"
	"github.com/aisc/pkg/statestore/test"
	"github.com/aisc/pkg/storage"
)

func TestMockStateStore(t *testing.T) {
	test.Run(t, func(t *testing.T) storage.StateStorer {
		t.Helper()
		return mock.NewStateStore()
	})
}
