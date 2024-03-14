// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package swap

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/aisc/pkg/storage"
	"github.com/aisc/pkg/ aisc"
)

var (
	peerPrefix            = "swap_chequebook_peer_"
	peerChequebookPrefix  = "swap_peer_chequebook_"
	beneficiaryPeerPrefix = "swap_beneficiary_peer_"
	peerBeneficiaryPrefix = "swap_peer_beneficiary_"
	deductedForPeerPrefix = "swap_deducted_for_peer_"
	deductedByPeerPrefix  = "swap_deducted_by_peer_"
)

// Addressbook maps peers to beneficaries, chequebooks and in reverse.
type Addressbook interface {
	// Beneficiary returns the beneficiary for the given peer.
	Beneficiary(peer  aisc.Address) (beneficiary common.Address, known bool, err error)
	// Chequebook returns the chequebook for the given peer.
	Chequebook(peer  aisc.Address) (chequebookAddress common.Address, known bool, err error)
	// BeneficiaryPeer returns the peer for a beneficiary.
	BeneficiaryPeer(beneficiary common.Address) (peer  aisc.Address, known bool, err error)
	// ChequebookPeer returns the peer for a beneficiary.
	ChequebookPeer(chequebook common.Address) (peer  aisc.Address, known bool, err error)
	// PutBeneficiary stores the beneficiary for the given peer.
	PutBeneficiary(peer  aisc.Address, beneficiary common.Address) error
	// PutChequebook stores the chequebook for the given peer.
	PutChequebook(peer  aisc.Address, chequebook common.Address) error
	// AddDeductionFor peer stores the flag indicating the peer have already issued a cheque that has aiscn deducted
	AddDeductionFor(peer  aisc.Address) error
	// AddDeductionFor peer stores the flag indicating the peer have already received a cheque that has aiscn deducted
	AddDeductionBy(peer  aisc.Address) error
	// GetDeductionFor returns whether a peer have already issued a cheque that has aiscn deducted
	GetDeductionFor(peer  aisc.Address) (bool, error)
	// GetDeductionBy returns whether a peer have already received a cheque that has aiscn deducted
	GetDeductionBy(peer  aisc.Address) (bool, error)
	// MigratePeer returns whether a peer have already received a cheque that has aiscn deducted
	MigratePeer(oldPeer, newPeer  aisc.Address) error
}

type addressbook struct {
	store storage.StateStorer
}

// NewAddressbook creates a new addressbook using the store.
func NewAddressbook(store storage.StateStorer) Addressbook {
	return &addressbook{
		store: store,
	}
}

func (a *addressbook) MigratePeer(oldPeer, newPeer  aisc.Address) error {
	ba, known, err := a.Beneficiary(oldPeer)
	if err != nil {
		return err
	}
	if !known {
		return errors.New("old beneficiary not known")
	}

	cb, known, err := a.Chequebook(oldPeer)
	if err != nil {
		return err
	}

	if err := a.PutBeneficiary(newPeer, ba); err != nil {
		return err
	}

	if err := a.store.Delete(peerBeneficiaryKey(oldPeer)); err != nil {
		return err
	}

	if known {
		if err := a.PutChequebook(newPeer, cb); err != nil {
			return err
		}
		if err := a.store.Delete(peerKey(oldPeer)); err != nil {
			return err
		}
	}

	return nil
}

// Beneficiary returns the beneficiary for the given peer.
func (a *addressbook) Beneficiary(peer  aisc.Address) (beneficiary common.Address, known bool, err error) {
	err = a.store.Get(peerBeneficiaryKey(peer), &beneficiary)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return common.Address{}, false, err
		}
		return common.Address{}, false, nil
	}
	return beneficiary, true, nil
}

// BeneficiaryPeer returns the peer for a beneficiary.
func (a *addressbook) BeneficiaryPeer(beneficiary common.Address) (peer  aisc.Address, known bool, err error) {
	err = a.store.Get(beneficiaryPeerKey(beneficiary), &peer)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return  aisc.Address{}, false, err
		}
		return  aisc.Address{}, false, nil
	}
	return peer, true, nil
}

// Chequebook returns the chequebook for the given peer.
func (a *addressbook) Chequebook(peer  aisc.Address) (chequebookAddress common.Address, known bool, err error) {
	err = a.store.Get(peerKey(peer), &chequebookAddress)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return common.Address{}, false, err
		}
		return common.Address{}, false, nil
	}
	return chequebookAddress, true, nil
}

// ChequebookPeer returns the peer for a beneficiary.
func (a *addressbook) ChequebookPeer(chequebook common.Address) (peer  aisc.Address, known bool, err error) {
	err = a.store.Get(chequebookPeerKey(chequebook), &peer)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return  aisc.Address{}, false, err
		}
		return  aisc.Address{}, false, nil
	}
	return peer, true, nil
}

// PutBeneficiary stores the beneficiary for the given peer.
func (a *addressbook) PutBeneficiary(peer  aisc.Address, beneficiary common.Address) error {
	err := a.store.Put(peerBeneficiaryKey(peer), beneficiary)
	if err != nil {
		return err
	}
	return a.store.Put(beneficiaryPeerKey(beneficiary), peer)
}

// PutChequebook stores the chequebook for the given peer.
func (a *addressbook) PutChequebook(peer  aisc.Address, chequebook common.Address) error {
	err := a.store.Put(peerKey(peer), chequebook)
	if err != nil {
		return err
	}
	return a.store.Put(chequebookPeerKey(chequebook), peer)
}

func (a *addressbook) AddDeductionFor(peer  aisc.Address) error {
	return a.store.Put(peerDeductedForKey(peer), struct{}{})
}

func (a *addressbook) AddDeductionBy(peer  aisc.Address) error {
	return a.store.Put(peerDeductedByKey(peer), struct{}{})
}

func (a *addressbook) GetDeductionFor(peer  aisc.Address) (bool, error) {
	var nothing struct{}
	err := a.store.Get(peerDeductedForKey(peer), &nothing)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (a *addressbook) GetDeductionBy(peer  aisc.Address) (bool, error) {
	var nothing struct{}
	err := a.store.Get(peerDeductedByKey(peer), &nothing)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// peerKey computes the key where to store the chequebook from a peer.
func peerKey(peer  aisc.Address) string {
	return fmt.Sprintf("%s%s", peerPrefix, peer)
}

// chequebookPeerKey computes the key where to store the peer for a chequebook.
func chequebookPeerKey(chequebook common.Address) string {
	return fmt.Sprintf("%s%x", peerChequebookPrefix, chequebook)
}

// peerBeneficiaryKey computes the key where to store the beneficiary for a peer.
func peerBeneficiaryKey(peer  aisc.Address) string {
	return fmt.Sprintf("%s%s", peerBeneficiaryPrefix, peer)
}

// beneficiaryPeerKey computes the key where to store the peer for a beneficiary.
func beneficiaryPeerKey(peer common.Address) string {
	return fmt.Sprintf("%s%x", beneficiaryPeerPrefix, peer)
}

func peerDeductedByKey(peer  aisc.Address) string {
	return fmt.Sprintf("%s%s", deductedByPeerPrefix, peer.String())
}

func peerDeductedForKey(peer  aisc.Address) string {
	return fmt.Sprintf("%s%s", deductedForPeerPrefix, peer.String())
}
