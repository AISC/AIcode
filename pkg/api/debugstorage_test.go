// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"net/http"
	"testing"

	"github.com/aisc/pkg/jsonhttp/jsonhttptest"
	"github.com/aisc/pkg/storer"
	mockstorer "github.com/aisc/pkg/storer/mock"
)

func TestDebugStorage(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()

		want := storer.Info{
			Upload: storer.UploadStat{
				TotalUploaded: 100,
				TotalSynced:   50,
			},
			Cache: storer.CacheStat{
				Size:     50,
				Capacity: 100,
			},
			ChunkStore: storer.ChunkStoreStat{
				TotalChunks: 100,
				SharedSlots: 10,
			},
		}

		ts, _, _, _ := newTestServer(t, testServerOptions{
			DebugAPI: true,
			Storer:   mockstorer.NewWithDebugInfo(want),
		})

		jsonhttptest.Request(t, ts, http.MethodGet, "/debugstore", http.StatusOK,
			jsonhttptest.WithExpectedJSONResponse(want),
		)
	})

}
