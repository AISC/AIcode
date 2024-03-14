// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	"github.com/aisc/pkg/api"
	"github.com/aisc/pkg/jsonhttp/jsonhttptest"
	"github.com/aisc/pkg/log"
	mockpost "github.com/aisc/pkg/postage/mock"
	resolverMock "github.com/aisc/pkg/resolver/mock"
	mockstorer "github.com/aisc/pkg/storer/mock"
	"github.com/aisc/pkg/ aisc"
)

func TestSubdomains(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name                string
		files               []f
		expectedReference    aisc.Address
		wantIndexFilename   string
		wantErrorFilename   string
		indexFilenameOption jsonhttptest.Option
		errorFilenameOption jsonhttptest.Option
	}{
		{
			name:              "nested files with extension",
			expectedReference:  aisc.MustParseHexAddress("4c9c76d63856102e54092c38a7cd227d769752d768b7adc8c3542e3dd9fcf295"),
			files: []f{
				{
					data: []byte("robots text"),
					name: "robots.txt",
					dir:  "",
					header: http.Header{
						api.ContentTypeHeader: {"text/plain; charset=utf-8"},
					},
				},
				{
					data: []byte("image 1"),
					name: "1.png",
					dir:  "img",
					header: http.Header{
						api.ContentTypeHeader: {"image/png"},
					},
				},
				{
					data: []byte("image 2"),
					name: "2.png",
					dir:  "img",
					header: http.Header{
						api.ContentTypeHeader: {"image/png"},
					},
				},
			},
		},
		{
			name:                "explicit index and error filename",
			expectedReference:    aisc.MustParseHexAddress("2cd9a6ac11eefbb71b372fb97c3ef64109c409955964a294fdc183c1014b3844"),
			wantIndexFilename:   "index.html",
			wantErrorFilename:   "error.html",
			indexFilenameOption: jsonhttptest.WithRequestHeader(api.AiscIndexDocumentHeader, "index.html"),
			errorFilenameOption: jsonhttptest.WithRequestHeader(api.AiscErrorDocumentHeader, "error.html"),
			files: []f{
				{
					data: []byte("<h1>Aisc"),
					name: "index.html",
					dir:  "",
					header: http.Header{
						api.ContentTypeHeader: {"text/html; charset=utf-8"},
					},
				},
				{
					data: []byte("<h2>404"),
					name: "error.html",
					dir:  "",
					header: http.Header{
						api.ContentTypeHeader: {"text/html; charset=utf-8"},
					},
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				dirUploadResource = "/aisc"
				storer            = mockstorer.New()
				logger            = log.Noop
				client, _, _, _   = newTestServer(t, testServerOptions{
					Storer:          storer,
					Logger:          logger,
					PreventRedirect: true,
					Post:            mockpost.New(mockpost.WithAcceptAll()),
					Resolver: resolverMock.NewResolver(
						resolverMock.WithResolveFunc(
							func(string) ( aisc.Address, error) {
								return tc.expectedReference, nil
							},
						),
					),
				})
			)

			validateAltPath := func(t *testing.T, fromPath, toPath string) {
				t.Helper()

				var respBytes []byte

				jsonhttptest.Request(t, client, http.MethodGet,
					fmt.Sprintf("http://test.eth. aisc.localhost/%s", toPath), http.StatusOK,
					jsonhttptest.WithPutResponseBody(&respBytes),
				)

				jsonhttptest.Request(t, client, http.MethodGet,
					fmt.Sprintf("http://test.eth. aisc.localhost/%s", fromPath), http.StatusOK,
					jsonhttptest.WithExpectedResponse(respBytes),
				)
			}

			tarReader := tarFiles(t, tc.files)

			var resp api. aiscUploadResponse

			options := []jsonhttptest.Option{
				jsonhttptest.WithRequestHeader(api.AiscDeferredUploadHeader, "true"),
				jsonhttptest.WithRequestHeader(api.AiscPostageBatchIdHeader, batchOkStr),
				jsonhttptest.WithRequestBody(tarReader),
				jsonhttptest.WithRequestHeader(api.AiscCollectionHeader, "True"),
				jsonhttptest.WithRequestHeader(api.ContentTypeHeader, api.ContentTypeTar),
				jsonhttptest.WithUnmarshalJSONResponse(&resp),
			}
			if tc.indexFilenameOption != nil {
				options = append(options, tc.indexFilenameOption)
			}
			if tc.errorFilenameOption != nil {
				options = append(options, tc.errorFilenameOption)
			}

			jsonhttptest.Request(t, client, http.MethodPost, dirUploadResource, http.StatusCreated, options...)

			if resp.Reference.String() == "" {
				t.Fatalf("expected file reference, did not got any")
			}

			if tc.expectedReference.String() != resp.Reference.String() {
				t.Fatalf("got unexpected reference exp %s got %s", tc.expectedReference.String(), resp.Reference.String())
			}

			for _, f := range tc.files {
				jsonhttptest.Request(
					t, client, http.MethodGet,
					fmt.Sprintf("http://test.eth. aisc.localhost/%s", path.Join(f.dir, f.name)),
					http.StatusOK,
					jsonhttptest.WithExpectedResponse(f.data),
				)
			}

			if tc.wantIndexFilename != "" {
				validateAltPath(t, "", tc.wantIndexFilename)
			}
			if tc.wantErrorFilename != "" {
				validateAltPath(t, "_non_existent_file_path_", tc.wantErrorFilename)
			}
		})
	}
}
