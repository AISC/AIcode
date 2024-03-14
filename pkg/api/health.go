// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"

	"github.com/aisc"
	"github.com/aisc/pkg/jsonhttp"
)

type healthStatusResponse struct {
	Status          string `json:"status"`
	Version         string `json:"version"`
	APIVersion      string `json:"apiVersion"`
	DebugAPIVersion string `json:"debugApiVersion"`
}

func (s *Service) healthHandler(w http.ResponseWriter, _ *http.Request) {
	status := s.probe.Healthy()
	jsonhttp.OK(w, healthStatusResponse{
		Status:          status.String(),
		Version:         aisc.Version,
		APIVersion:      Version,
		DebugAPIVersion: DebugVersion,
	})
}
