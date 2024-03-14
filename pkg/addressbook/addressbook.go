// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package addressbook

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aisc/pkg/aisc"
	"github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
)

const keyPrefix = "addressbook_entry_"

var _ Interface = (*store)(nil)

var ErrNotFound = errors.New("addressbook: not found")

// Interface is the AddressBook interface.
type Interface interface {
	GetPutter
	Remover
	// Overlays returns a list of all overlay addresses saved in addressbook.
	Overlays() ([] aisc.Address, error)
	// IterateOverlays exposes overlays in a form of an iterator.
	IterateOverlays(func( aisc.Address) (bool, error)) error
	// Addresses returns a list of all aisc.Address-es saved in addressbook.
	Addresses() ([]aisc.Address, error)
}

type GetPutter interface {
	Getter
	Putter
}

type Getter interface {
	// Get returns pointer to saved aisc.Address for requested overlay address.
	Get(overlay  aisc.Address) (addr *aisc.Address, err error)
}

type Putter interface {
	// Put saves relation between peer overlay address and aisc.Address address.
	Put(overlay  aisc.Address, addr aisc.Address) (err error)
}

type Remover interface {
	// Remove removes overlay address.
	Remove(overlay  aisc.Address) error
}

type store struct {
	store storage.StateStorer
}

// New creates new addressbook for state storer.
func New(storer storage.StateStorer) Interface {
	return &store{
		store: storer,
	}
}

func (s *store) Get(overlay  aisc.Address) (*aisc.Address, error) {
	key := keyPrefix + overlay.String()
	v := &aisc.Address{}
	err := s.store.Get(key, &v)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}
	return v, nil
}

func (s *store) Put(overlay  aisc.Address, addr aisc.Address) (err error) {
	key := keyPrefix + overlay.String()
	return s.store.Put(key, &addr)
}

func (s *store) Remove(overlay  aisc.Address) error {
	return s.store.Delete(keyPrefix + overlay.String())
}

func (s *store) IterateOverlays(cb func( aisc.Address) (bool, error)) error {
	return s.store.Iterate(keyPrefix, func(key, _ []byte) (stop bool, err error) {
		k := string(key)
		if !strings.HasPrefix(k, keyPrefix) {
			return true, nil
		}
		split := strings.SplitAfter(k, keyPrefix)
		if len(split) != 2 {
			return true, fmt.Errorf("invalid overlay key: %s", k)
		}
		addr, err :=  aisc.ParseHexAddress(split[1])
		if err != nil {
			return true, err
		}
		stop, err = cb(addr)
		if err != nil {
			return true, err
		}
		if stop {
			return true, nil
		}
		return false, nil
	})
}

func (s *store) Overlays() (overlays [] aisc.Address, err error) {
	err = s.IterateOverlays(func(addr  aisc.Address) (stop bool, err error) {
		overlays = append(overlays, addr)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return overlays, nil
}

func (s *store) Addresses() (addresses []aisc.Address, err error) {
	err = s.store.Iterate(keyPrefix, func(_, value []byte) (stop bool, err error) {
		entry := &aisc.Address{}
		err = entry.UnmarshalJSON(value)
		if err != nil {
			return true, err
		}

		addresses = append(addresses, *entry)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return addresses, nil
}
