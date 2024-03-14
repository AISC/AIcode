// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nbhdutil

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"

	"github.com/aisc/pkg/crypto"
	"github.com/aisc/pkg/ aisc"
)

func MineOverlay(ctx context.Context, p ecdsa.PublicKey, networkID uint64, targetNeighborhood string) ( aisc.Address, []byte, error) {

	nonce := make([]byte, 32)

	neighborhood, err :=  aisc.ParseBitStrAddress(targetNeighborhood)
	if err != nil {
		return  aisc.ZeroAddress, nil, err
	}
	prox := len(targetNeighborhood)

	i := uint64(0)
	for {

		select {
		case <-ctx.Done():
			return  aisc.ZeroAddress, nil, ctx.Err()
		default:
		}

		binary.LittleEndian.PutUint64(nonce, i)

		 aiscAddress, err := crypto.NewOverlayAddress(p, networkID, nonce)
		if err != nil {
			return  aisc.ZeroAddress, nil, fmt.Errorf("compute overlay address: %w", err)
		}

		if  aisc.Proximity( aiscAddress.Bytes(), neighborhood.Bytes()) >= uint8(prox) {
			return  aiscAddress, nonce, nil
		}

		i++
	}
}
