// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"context"
	"math/big"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/aisc/pkg/api"
	"github.com/aisc/pkg/bigint"
	"github.com/aisc/pkg/jsonhttp"
	"github.com/aisc/pkg/jsonhttp/jsonhttptest"
	erc20mock "github.com/aisc/pkg/settlement/swap/erc20/mock"
	"github.com/aisc/pkg/transaction/backendmock"
)

func TestWallet(t *testing.T) {
	t.Parallel()

	t.Run("Okay", func(t *testing.T) {
		t.Parallel()

		srv, _, _, _ := newTestServer(t, testServerOptions{
			DebugAPI: true,
			Erc20Opts: []erc20mock.Option{
				erc20mock.WithBalanceOfFunc(func(ctx context.Context, address common.Address) (*big.Int, error) {
					return big.NewInt(10000000000000000), nil
				}),
			},
			BackendOpts: []backendmock.Option{
				backendmock.WithBalanceAt(func(ctx context.Context, address common.Address, block *big.Int) (*big.Int, error) {
					return big.NewInt(2000000000000000000), nil
				}),
			},
		})

		jsonhttptest.Request(t, srv, http.MethodGet, "/wallet", http.StatusOK,
			jsonhttptest.WithExpectedJSONResponse(api.WalletResponse{
				 aisc:         bigint.Wrap(big.NewInt(10000000000000000)),
				NativeToken: bigint.Wrap(big.NewInt(2000000000000000000)),
				ChainID:     1,
			}),
		)
	})

	t.Run("500 - erc20 error", func(t *testing.T) {
		t.Parallel()

		srv, _, _, _ := newTestServer(t, testServerOptions{
			DebugAPI: true,
			BackendOpts: []backendmock.Option{
				backendmock.WithBalanceAt(func(ctx context.Context, address common.Address, block *big.Int) (*big.Int, error) {
					return new(big.Int), nil
				}),
			},
		})

		jsonhttptest.Request(t, srv, http.MethodGet, "/wallet", http.StatusInternalServerError,
			jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
				Message: "unable to acquire erc20 balance",
				Code:    500,
			}))
	})

	t.Run("500 - chain backend error", func(t *testing.T) {
		t.Parallel()

		srv, _, _, _ := newTestServer(t, testServerOptions{
			DebugAPI: true,
			Erc20Opts: []erc20mock.Option{
				erc20mock.WithBalanceOfFunc(func(ctx context.Context, address common.Address) (*big.Int, error) {
					return new(big.Int), nil
				}),
			},
		})

		jsonhttptest.Request(t, srv, http.MethodGet, "/wallet", http.StatusInternalServerError,
			jsonhttptest.WithExpectedJSONResponse(jsonhttp.StatusResponse{
				Message: "unable to acquire balance from the chain backend",
				Code:    500,
			}))
	})
}
