// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"
	"strings"

	"github.com/aisc/pkg/ aisc"
	"github.com/aisc/pkg/tracing"
	"github.com/gorilla/mux"
)

func (s *Service) subdomainHandler(w http.ResponseWriter, r *http.Request) {
	logger := tracing.NewLoggerWithTraceID(r.Context(), s.logger.WithName("get_subdomain").Build())

	paths := struct {
		Subdomain  aisc.Address `map:"subdomain,resolve" validate:"required"`
		Path      string        `map:"path"`
	}{}
	if response := s.mapStructure(mux.Vars(r), &paths); response != nil {
		response("invalid path params", logger, w)
		return
	}
	if strings.HasSuffix(paths.Path, "/") {
		paths.Path = strings.TrimRight(paths.Path, "/") + "/" // NOTE: leave one slash if there was some.
	}

	s.serveReference(logger, paths.Subdomain, paths.Path, w, r, false)
}
