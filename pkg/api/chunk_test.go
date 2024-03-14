// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/aisc/pkg/log"
	mockbatchstore "github.com/aisc/pkg/postage/batchstore/mock"
	mockpost "github.com/aisc/pkg/postage/mock"
	"github.com/aisc/pkg/spinlock"
	mockstorer "github.com/aisc/pkg/storer/mock"

	"github.com/aisc/pkg/api"
	"github.com/aisc/pkg/jsonhttp"
	"github.com/aisc/pkg/jsonhttp/jsonhttptest"
	testingc "github.com/aisc/pkg/storage/testing"
	"github.com/aisc/pkg/ aisc"
)

// nolint:paralleltest,tparallel
// TestChunkUploadDownload uploads a chunk to an API that verifies the chunk according
// to a given validator, then tries to download the uploaded data.
func TestChunkUploadDownload(t *testing.T) {
	t.Parallel()

	var (
		chunksEndpoint           = "/chunks"
		chunksResource           = func(a  aisc.Address) string { return "/chunks/" + a.String() }
		chunk                    = testingc.GenerateTestRandomChunk()
		storerMock               = mockstorer.New()
		client, _, _, chanStorer = newTestServer(t, testServerOptions{
			Storer:       storerMock,
			Post:         mockpost.New(mockpost.WithAcceptAll()),
			DirectUpload: true,
		})
	)

	t.Run("empty chunk", func(t *testing.T) {
		jsonhttptest.Request(t, client, http.MethodPost, chunksEndpoint, http.StatusBadRequest,
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
				Message: "insufficient data length",
				Code:    http.StatusBadRequest,
			}),
		)
	})

	t.Run("ok", func(t *testing.T) {
		tag, err := storerMock.NewSession()
		if err != nil {
			t.Fatalf("failed creating tag: %v", err)
		}

		jsonhttptest.Request(t, client, http.MethodPost, chunksEndpoint, http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.AiscTagHeader, fmt.Sprintf("%d", tag.TagID)),
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(bytes.NewReader(chunk.Data())),
			jsonhttptest.WithExpectedJSONResponse(api.ChunkAddressResponse{Reference: chunk.Address()}),
		)

		has, err := storerMock.ChunkStore().Has(context.Background(), chunk.Address())
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Fatal("storer check root chunk reference: have none; want one")
		}

		// try to fetch the same chunk
		jsonhttptest.Request(t, client, http.MethodGet, chunksResource(chunk.Address()), http.StatusOK,
			jsonhttptest.WithExpectedResponse(chunk.Data()),
			jsonhttptest.WithExpectedContentLength(len(chunk.Data())),
		)
	})

	t.Run("direct upload ok", func(t *testing.T) {
		jsonhttptest.Request(t, client, http.MethodPost, chunksEndpoint, http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(bytes.NewReader(chunk.Data())),
			jsonhttptest.WithExpectedJSONResponse(api.ChunkAddressResponse{Reference: chunk.Address()}),
		)

		time.Sleep(time.Millisecond * 100)
		err := spinlock.Wait(time.Second, func() bool { return chanStorer.Has(chunk.Address()) })
		if err != nil {
			t.Fatal(err)
		}
	})
}

// nolint:paralleltest,tparallel
func TestChunkHasHandler(t *testing.T) {
	mockStorer := mockstorer.New()
	testServer, _, _, _ := newTestServer(t, testServerOptions{
		Storer: mockStorer,
	})

	key :=  aisc.MustParseHexAddress("aabbcc")
	value := []byte("data data data")

	err := mockStorer.Cache().Put(context.Background(),  aisc.NewChunk(key, value))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ok", func(t *testing.T) {
		jsonhttptest.Request(t, testServer, http.MethodHead, "/chunks/"+key.String(), http.StatusOK,
			jsonhttptest.WithNoResponseBody(),
		)

		jsonhttptest.Request(t, testServer, http.MethodGet, "/chunks/"+key.String(), http.StatusOK,
			jsonhttptest.WithExpectedResponse(value),
			jsonhttptest.WithExpectedContentLength(len(value)),
		)
	})

	t.Run("not found", func(t *testing.T) {
		jsonhttptest.Request(t, testServer, http.MethodHead, "/chunks/abbbbb", http.StatusNotFound,
			jsonhttptest.WithNoResponseBody())
	})

	t.Run("bad address", func(t *testing.T) {
		jsonhttptest.Request(t, testServer, http.MethodHead, "/chunks/abcd1100zz", http.StatusBadRequest,
			jsonhttptest.WithNoResponseBody())
	})
}

func TestChunkHandlersInvalidInputs(t *testing.T) {
	t.Parallel()

	client, _, _, _ := newTestServer(t, testServerOptions{})

	tests := []struct {
		name    string
		address string
		want    jsonhttp.StatusResponse
	}{{
		name:    "address odd hex string",
		address: "123",
		want: jsonhttp.StatusResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid path params",
			Reasons: []jsonhttp.Reason{
				{
					Field: "address",
					Error: api.ErrHexLength.Error(),
				},
			},
		},
	}, {
		name:    "address invalid hex character",
		address: "123G",
		want: jsonhttp.StatusResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid path params",
			Reasons: []jsonhttp.Reason{
				{
					Field: "address",
					Error: api.HexInvalidByteError('G').Error(),
				},
			},
		},
	}}

	method := http.MethodGet
	for _, tc := range tests {
		tc := tc
		t.Run(method+" "+tc.name, func(t *testing.T) {
			t.Parallel()

			jsonhttptest.Request(t, client, method, "/chunks/"+tc.address, tc.want.Code,
				jsonhttptest.WithExpectedJSONResponse(tc.want),
			)
		})
	}
}

func TestChunkInvalidParams(t *testing.T) {
	t.Parallel()

	var (
		chunksEndpoint = "/chunks"
		chunk          = testingc.GenerateTestRandomChunk()
		storerMock     = mockstorer.New()
		logger         = log.Noop
		existsFn       = func(id []byte) (bool, error) {
			return false, errors.New("error")
		}
	)

	t.Run("batch unusable", func(t *testing.T) {
		t.Parallel()

		clientBatchUnusable, _, _, _ := newTestServer(t, testServerOptions{
			Storer:     storerMock,
			Logger:     logger,
			Post:       mockpost.New(mockpost.WithAcceptAll()),
			BatchStore: mockbatchstore.New(),
		})
		jsonhttptest.Request(t, clientBatchUnusable, http.MethodPost, chunksEndpoint, http.StatusUnprocessableEntity,
			jsonhttptest.WithRequestHeader(api.AiscDeferredUploadHeader, "true"),
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(bytes.NewReader(chunk.Data())),
		)
	})

	t.Run("batch exists", func(t *testing.T) {
		t.Parallel()

		clientBatchExists, _, _, _ := newTestServer(t, testServerOptions{
			Storer:     storerMock,
			Logger:     logger,
			Post:       mockpost.New(mockpost.WithAcceptAll()),
			BatchStore: mockbatchstore.New(mockbatchstore.WithExistsFunc(existsFn)),
		})
		jsonhttptest.Request(t, clientBatchExists, http.MethodPost, chunksEndpoint, http.StatusBadRequest,
			jsonhttptest.WithRequestHeader(api.AiscDeferredUploadHeader, "true"),
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(bytes.NewReader(chunk.Data())),
		)
	})

	t.Run("batch not found", func(t *testing.T) {
		t.Parallel()

		clientBatchNotFound, _, _, _ := newTestServer(t, testServerOptions{
			Storer: storerMock,
			Logger: logger,
			Post:   mockpost.New(),
		})
		jsonhttptest.Request(t, clientBatchNotFound, http.MethodPost, chunksEndpoint, http.StatusNotFound,
			jsonhttptest.WithRequestHeader(api.AiscDeferredUploadHeader, "true"),
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(bytes.NewReader(chunk.Data())),
		)
	})
}

// // TestDirectChunkUpload tests that the direct upload endpoint give correct error message in dev mode
func TestChunkDirectUpload(t *testing.T) {
	t.Parallel()
	var (
		chunksEndpoint  = "/chunks"
		chunk           = testingc.GenerateTestRandomChunk()
		storerMock      = mockstorer.New()
		client, _, _, _ = newTestServer(t, testServerOptions{
			Storer:  storerMock,
			Post:    mockpost.New(mockpost.WithAcceptAll()),
			AiscMode: api.DevMode,
		})
	)

	jsonhttptest.Request(t, client, http.MethodPost, chunksEndpoint, http.StatusBadRequest,
		jsonhttptest.WithRequestHeader(api.AiscDeferredUploadHeader, "false"),
		jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
		jsonhttptest.WithRequestBody(bytes.NewReader(chunk.Data())),
		jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
			Message: api.ErrUnsupportedDevNodeOperation.Error(),
			Code:    http.StatusBadRequest,
		}),
	)
}
