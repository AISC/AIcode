// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pinstore

import (
	"fmt"

	storage "github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
)

type (
	PinCollectionItem = pinCollectionItem
	PinChunkItem      = pinChunkItem
	DirtyCollection   = dirtyCollection
)

var (
	ErrInvalidPinCollectionItemAddr = errInvalidPinCollectionAddr
	ErrInvalidPinCollectionItemUUID = errInvalidPinCollectionUUID
	ErrInvalidPinCollectionItemSize = errInvalidPinCollectionSize
	ErrPutterAlreadyClosed          = errPutterAlreadyClosed
	ErrCollectionRootAddressIsZero  = errCollectionRootAddressIsZero
	ErrDuplicatePinCollection       = errDuplicatePinCollection
)

var NewUUID = newUUID

func GetStat(st storage.Store, root  aisc.Address) (CollectionStat, error) {
	collection := &pinCollectionItem{Addr: root}
	err := st.Get(collection)
	if err != nil {
		return CollectionStat{}, fmt.Errorf("pin store: failed getting collection: %w", err)
	}

	return collection.Stat, nil
}
