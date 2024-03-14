// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chunkstamp

import "github.com/aisc/pkg/ aisc"

type Item = item

func (i *Item) WithNamespace(ns string) *Item {
	i.namespace = []byte(ns)
	return i
}

func (i *Item) WithAddress(addr  aisc.Address) *Item {
	i.address = addr
	return i
}

func (i *Item) WithStamp(stamp  aisc.Stamp) *Item {
	i.stamp = stamp
	return i
}

var (
	ErrMarshalInvalidChunkStampItemNamespace = errMarshalInvalidChunkStampItemNamespace
	ErrMarshalInvalidChunkStampItemAddress   = errMarshalInvalidChunkStampItemAddress
	ErrUnmarshalInvalidChunkStampItemAddress = errUnmarshalInvalidChunkStampItemAddress
	ErrMarshalInvalidChunkStampItemStamp     = errMarshalInvalidChunkStampItemStamp
	ErrUnmarshalInvalidChunkStampItemSize    = errUnmarshalInvalidChunkStampItemSize
)
