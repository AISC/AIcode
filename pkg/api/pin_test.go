// Copyright 2021 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aisc/pkg/api"
	"github.com/aisc/pkg/jsonhttp"
	"github.com/aisc/pkg/jsonhttp/jsonhttptest"
	mockpost "github.com/aisc/pkg/postage/mock"
	mockstorer "github.com/aisc/pkg/storer/mock"
	"github.com/aisc/pkg/ aisc"
)

func checkPinHandlers(t *testing.T, client *http.Client, rootHash string, createPin bool) {
	t.Helper()

	const pinsBasePath = "/pins"

	var (
		pinsReferencePath        = pinsBasePath + "/" + rootHash
		pinsInvalidReferencePath = pinsBasePath + "/" + "838d0a193ecd1152d1bb1432d5ecc02398533b2494889e23b8bd5ace30ac2zzz"
		pinsUnknownReferencePath = pinsBasePath + "/" + "838d0a193ecd1152d1bb1432d5ecc02398533b2494889e23b8bd5ace30ac2ccc"
	)

	jsonhttptest.Request(t, client, http.MethodGet, pinsInvalidReferencePath, http.StatusBadRequest)

	jsonhttptest.Request(t, client, http.MethodGet, pinsUnknownReferencePath, http.StatusNotFound,
		jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
			Message: http.StatusText(http.StatusNotFound),
			Code:    http.StatusNotFound,
		}),
	)

	if createPin {
		jsonhttptest.Request(t, client, http.MethodPost, pinsReferencePath, http.StatusCreated,
			jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
				Message: http.StatusText(http.StatusCreated),
				Code:    http.StatusCreated,
			}),
		)
	}

	jsonhttptest.Request(t, client, http.MethodGet, pinsReferencePath, http.StatusOK,
		jsonhttptest.WithExpectedJSONResponse(struct {
			Reference  aisc.Address `json:"reference"`
		}{
			Reference:  aisc.MustParseHexAddress(rootHash),
		}),
	)

	jsonhttptest.Request(t, client, http.MethodGet, pinsBasePath, http.StatusOK,
		jsonhttptest.WithExpectedJSONResponse(struct {
			References [] aisc.Address `json:"references"`
		}{
			References: [] aisc.Address{ aisc.MustParseHexAddress(rootHash)},
		}),
	)

	jsonhttptest.Request(t, client, http.MethodDelete, pinsReferencePath, http.StatusOK)

	jsonhttptest.Request(t, client, http.MethodGet, pinsReferencePath, http.StatusNotFound,
		jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
			Message: http.StatusText(http.StatusNotFound),
			Code:    http.StatusNotFound,
		}),
	)
}

// nolint:paralleltest
func TestPinHandlers(t *testing.T) {
	var (
		storerMock      = mockstorer.New()
		client, _, _, _ = newTestServer(t, testServerOptions{
			Storer: storerMock,
			Post:   mockpost.New(mockpost.WithAcceptAll()),
		})
	)

	t.Run("bytes", func(t *testing.T) {
		const rootHash = "838d0a193ecd1152d1bb1432d5ecc02398533b2494889e23b8bd5ace30ac2aeb"
		jsonhttptest.Request(t, client, http.MethodPost, "/bytes", http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.AiscDeferredUploadHeader, "true"),
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(strings.NewReader("this is a simple text")),
			jsonhttptest.WithExpectedJSONResponse(api. aiscUploadResponse{
				Reference:  aisc.MustParseHexAddress(rootHash),
			}),
		)
		checkPinHandlers(t, client, rootHash, true)
	})

	t.Run("bytes missing", func(t *testing.T) {
		jsonhttptest.Request(t, client, http.MethodPost, "/pins/"+ aisc.RandAddress(t).String(), http.StatusNotFound)
	})

	t.Run("aisc", func(t *testing.T) {
		tarReader := tarFiles(t, []f{{
			data: []byte("<h1>Aisc"),
			name: "index.html",
			dir:  "",
		}})
		rootHash := "9e178dbd1ed4b748379e25144e28dfb29c07a4b5114896ef454480115a56b237"
		jsonhttptest.Request(t, client, http.MethodPost, "/aisc", http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.AiscDeferredUploadHeader, "true"),
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestBody(tarReader),
			jsonhttptest.WithRequestHeader(api.ContentTypeHeader, api.ContentTypeTar),
			jsonhttptest.WithRequestHeader(api.AiscCollectionHeader, "true"),
			jsonhttptest.WithRequestHeader(api.AiscPinHeader, "true"),
			jsonhttptest.WithExpectedJSONResponse(api. aiscUploadResponse{
				Reference:  aisc.MustParseHexAddress(rootHash),
			}),
		)
		checkPinHandlers(t, client, rootHash, false)

		header := jsonhttptest.Request(t, client, http.MethodPost, "/aisc?name=somefile.txt", http.StatusCreated,
			jsonhttptest.WithRequestHeader(api.AiscDeferredUploadHeader, "true"),
			jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
			jsonhttptest.WithRequestHeader(api.ContentTypeHeader, "text/plain"),
			jsonhttptest.WithRequestHeader(api.AiscEncryptHeader, "true"),
			jsonhttptest.WithRequestHeader(api.AiscPinHeader, "true"),
			jsonhttptest.WithRequestBody(strings.NewReader("this is a simple text")),
		)

		rootHash = strings.Trim(header.Get(api.ETagHeader), "\"")
		checkPinHandlers(t, client, rootHash, false)
	})

}

func TestPinHandlersInvalidInputs(t *testing.T) {
	t.Parallel()

	client, _, _, _ := newTestServer(t, testServerOptions{})

	tests := []struct {
		name      string
		reference string
		want      jsonhttp.StatusResponse
	}{{
		name:      "reference - odd hex string",
		reference: "123",
		want: jsonhttp.StatusResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid path params",
			Reasons: []jsonhttp.Reason{
				{
					Field: "reference",
					Error: api.ErrHexLength.Error(),
				},
			},
		},
	}, {
		name:      "reference - invalid hex character",
		reference: "123G",
		want: jsonhttp.StatusResponse{
			Code:    http.StatusBadRequest,
			Message: "invalid path params",
			Reasons: []jsonhttp.Reason{
				{
					Field: "reference",
					Error: api.HexInvalidByteError('G').Error(),
				},
			},
		},
	}}

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		method := method
		for _, tc := range tests {
			tc := tc
			t.Run(method+" "+tc.name, func(t *testing.T) {
				t.Parallel()

				jsonhttptest.Request(t, client, method, "/pins/"+tc.reference, tc.want.Code,
					jsonhttptest.WithExpectedJSONResponse(tc.want),
				)
			})
		}
	}
}
