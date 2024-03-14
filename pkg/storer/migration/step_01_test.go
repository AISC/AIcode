// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package migration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aisc/pkg/storage/inmemstore"
	localmigration "github.com/aisc/pkg/storer/migration"
)

func Test_Step_01(t *testing.T) {
	t.Parallel()

	stepFn := localmigration.Step_01
	store := inmemstore.New()

	assert.NoError(t, stepFn(store))
}
