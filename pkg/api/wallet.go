// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/aisc/pkg/bigint"
	"github.com/aisc/pkg/jsonhttp"
)

type walletResponse struct {
	 aisc                       *bigint.BigInt `json:"aiscBalance"`                // the  aisc balance of the wallet associated with the eth address of the node
	NativeToken               *bigint.BigInt `json:"nativeTokenBalance"`        // the native token balance of the wallet associated with the eth address of the node
	ChainID                   int64          `json:"chainID"`                   // the id of the blockchain
	ChequebookContractAddress common.Address `json:"chequebookContractAddress"` // the address of the chequebook contract
	WalletAddress             common.Address `json:"walletAddress"`             // the address of the aisc wallet
}

func (s *Service) walletHandler(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.WithName("get_wallet").Build()

	nativeToken, err := s.chainBackend.BalanceAt(r.Context(), s.ethereumAddress, nil)
	if err != nil {
		logger.Debug("unable to acquire balance from the chain backend", "error", err)
		logger.Error(nil, "unable to acquire balance from the chain backend")
		jsonhttp.InternalServerError(w, "unable to acquire balance from the chain backend")
		return
	}

	aisc, err := s.erc20Service.BalanceOf(r.Context(), s.ethereumAddress)
	if err != nil {
		logger.Debug("unable to acquire erc20 balance", "error", err)
		logger.Error(nil, "unable to acquire erc20 balance")
		jsonhttp.InternalServerError(w, "unable to acquire erc20 balance")
		return
	}

	jsonhttp.OK(w, walletResponse{
		 aisc:                       bigint.Wrap(aisc),
		NativeToken:               bigint.Wrap(nativeToken),
		ChainID:                   s.chainID,
		ChequebookContractAddress: s.chequebook.Address(),
		WalletAddress:             s.ethereumAddress,
	})
}
