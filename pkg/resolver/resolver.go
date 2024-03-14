// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resolver

import (
	"errors"
	"io"

	"github.com/aisc/pkg/ aisc"
)

// Address is the  aisc aisc address.
type Address =  aisc.Address

var (
	// ErrParse denotes failure to parse given value
	ErrParse = errors.New("could not parse")
	// ErrNotFound denotes that given name was not found
	ErrNotFound = errors.New("not found")
	// ErrServiceNotAvailable denotes that remote ENS service is not available
	ErrServiceNotAvailable = errors.New("not available")
	// ErrInvalidContentHash denotes that the value of the response contenthash record is not valid.
	ErrInvalidContentHash = errors.New("invalid  aisc content hash")
)

// Interface can resolve an URL into an associated Ethereum address.
type Interface interface {
	Resolve(url string) (Address, error)
	io.Closer
}
