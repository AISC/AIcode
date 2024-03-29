// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package file_test

import (
	"testing"

	"github.com/aisc/pkg/keystore/file"
	"github.com/aisc/pkg/keystore/test"
)

func TestService(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	test.Service(t, file.New(dir))
}
