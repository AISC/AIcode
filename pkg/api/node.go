// Copyright 2021 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"

	"github.com/aisc/pkg/jsonhttp"
)

type AiscNodeMode uint

const (
	UnknownMode AiscNodeMode = iota
	LightMode
	FullMode
	DevMode
	UltraLightMode
)

type nodeResponse struct {
	AiscMode           string `json:"aiscMode"`
	ChequebookEnabled bool   `json:"chequebookEnabled"`
	SwapEnabled       bool   `json:"swapEnabled"`
}

func (b AiscNodeMode) String() string {
	switch b {
	case LightMode:
		return "light"
	case FullMode:
		return "full"
	case DevMode:
		return "dev"
	case UltraLightMode:
		return "ultra-light"
	}
	return "unknown"
}

// nodeGetHandler gives back information about the Aisc node configuration.
func (s *Service) nodeGetHandler(w http.ResponseWriter, _ *http.Request) {
	jsonhttp.OK(w, nodeResponse{
		AiscMode:           s.aiscMode.String(),
		ChequebookEnabled: s.chequebookEnabled,
		SwapEnabled:       s.swapEnabled,
	})
}
