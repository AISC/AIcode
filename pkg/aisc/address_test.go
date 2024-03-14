// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aisc_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/aisc/pkg/aisc"
	"github.com/aisc/pkg/crypto"

	ma "github.com/multiformats/go-multiaddr"
)

func Test aiscAddress(t *testing.T) {
	t.Parallel()

	node1ma, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/1634/p2p/16Uiu2HAkx8ULY8cTXhdVAcMmLcH9AsTKz6uBQ7DPLKRjMLgBVYkA")
	if err != nil {
		t.Fatal(err)
	}

	nonce := common.HexToHash("0x2").Bytes()

	privateKey1, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		t.Fatal(err)
	}

	overlay, err := crypto.NewOverlayAddress(privateKey1.PublicKey, 3, nonce)
	if err != nil {
		t.Fatal(err)
	}
	signer1 := crypto.NewDefaultSigner(privateKey1)

	aiscAddress, err := aisc.NewAddress(signer1, node1ma, overlay, 3, nonce)
	if err != nil {
		t.Fatal(err)
	}

	aiscAddress2, err := aisc.ParseAddress(node1ma.Bytes(), overlay.Bytes(), aiscAddress.Signature, nonce, true, 3)
	if err != nil {
		t.Fatal(err)
	}

	if !aiscAddress.Equal(aiscAddress2) {
		t.Fatalf("got %s expected %s", aiscAddress2, aiscAddress)
	}

	bytes, err := aiscAddress.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	var newaisc aisc.Address
	if err := newaisc.UnmarshalJSON(bytes); err != nil {
		t.Fatal(err)
	}

	if !newaisc.Equal(aiscAddress) {
		t.Fatalf("got %s expected %s", newaisc, aiscAddress)
	}
}
