// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"

	"github.com/aisc/pkg/jsonhttp"
	"github.com/aisc/pkg/ aisc"
	"github.com/gorilla/mux"
)

func (s *Service) hasChunkHandler(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.WithName("get_chunk").Build()

	paths := struct {
		Address  aisc.Address `map:"address" validate:"required"`
	}{}
	if response := s.mapStructure(mux.Vars(r), &paths); response != nil {
		response("invalid path params", logger, w)
		return
	}

	has, err := s.storer.ChunkStore().Has(r.Context(), paths.Address)
	if err != nil {
		logger.Debug("has chunk failed", "chunk_address", paths.Address, "error", err)
		jsonhttp.BadRequest(w, err)
		return
	}

	if !has {
		jsonhttp.NotFound(w, nil)
		return
	}
	jsonhttp.OK(w, nil)
}
