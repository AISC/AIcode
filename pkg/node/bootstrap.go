// Copyright 2022 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package node

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/aisc/pkg/accounting"
	"github.com/aisc/pkg/addressbook"
	"github.com/aisc/pkg/crypto"
	"github.com/aisc/pkg/feeds"
	"github.com/aisc/pkg/feeds/factory"
	"github.com/aisc/pkg/file"
	"github.com/aisc/pkg/file/joiner"
	"github.com/aisc/pkg/file/loadsave"
	"github.com/aisc/pkg/hive"
	"github.com/aisc/pkg/log"
	"github.com/aisc/pkg/manifest"
	"github.com/aisc/pkg/p2p/libp2p"
	"github.com/aisc/pkg/postage"
	"github.com/aisc/pkg/pricer"
	"github.com/aisc/pkg/pricing"
	"github.com/aisc/pkg/retrieval"
	"github.com/aisc/pkg/settlement/pseudosettle"
	"github.com/aisc/pkg/settlement/swap/chequebook"
	"github.com/aisc/pkg/spinlock"
	"github.com/aisc/pkg/storage"
	storer "github.com/aisc/pkg/storer"
	"github.com/aisc/pkg/ aisc"
	"github.com/aisc/pkg/topology"
	"github.com/aisc/pkg/topology/kademlia"
	"github.com/aisc/pkg/topology/lightnode"
	"github.com/aisc/pkg/tracing"
	"github.com/aisc/pkg/transaction"
	"github.com/hashicorp/go-multierror"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	// zeroed out while waiting to be  replacement for the new snapshot feed address
	// must be different to avoid stale reads on the old contract
	snapshotFeed    =  aisc.MustParseHexAddress("0000000000000000000000000000000000000000000000000000000000000000")
	errDataMismatch = errors.New("data length mismatch")
)

const (
	getSnapshotRetries = 3
	retryWait          = time.Second * 5
	timeout            = time.Minute * 2
)

func bootstrapNode(
	ctx context.Context,
	addr string,
	 aiscAddress  aisc.Address,
	nonce []byte,
	chainID int64,
	overlayEthAddress common.Address,
	addressbook addressbook.Interface,
	bootnodes []ma.Multiaddr,
	lightNodes *lightnode.Container,
	chequebookService chequebook.Service,
	chequeStore chequebook.ChequeStore,
	cashoutService chequebook.CashoutService,
	transactionService transaction.Service,
	stateStore storage.StateStorer,
	signer crypto.Signer,
	networkID uint64,
	logger log.Logger,
	libp2pPrivateKey *ecdsa.PrivateKey,
	o *Options,
) (snapshot *postage.ChainSnapshot, retErr error) {

	tracer, tracerCloser, err := tracing.NewTracer(&tracing.Options{
		Enabled:     o.TracingEnabled,
		Endpoint:    o.TracingEndpoint,
		ServiceName: o.TracingServiceName,
	})
	if err != nil {
		return nil, fmt.Errorf("tracer: %w", err)
	}

	p2pCtx, p2pCancel := context.WithCancel(ctx)

	b := &Aisc{
		ctxCancel:    p2pCancel,
		tracerCloser: tracerCloser,
	}

	defer func() {
		retErr = multierror.Append(new(multierror.Error), retErr, b.Shutdown()).ErrorOrNil()
	}()

	p2ps, err := libp2p.New(p2pCtx, signer, networkID,  aiscAddress, addr, addressbook, stateStore, lightNodes, logger, tracer, libp2p.Options{
		PrivateKey:     libp2pPrivateKey,
		NATAddr:        o.NATAddr,
		EnableWS:       o.EnableWS,
		WelcomeMessage: o.WelcomeMessage,
		FullNode:       false,
		Nonce:          nonce,
	})
	if err != nil {
		return nil, fmt.Errorf("p2p service: %w", err)
	}
	b.p2pService = p2ps
	b.p2pHalter = p2ps

	hive := hive.New(p2ps, addressbook, networkID, o.BootnodeMode, o.AllowPrivateCIDRs, logger)

	if err = p2ps.AddProtocol(hive.Protocol()); err != nil {
		return nil, fmt.Errorf("hive service: %w", err)
	}
	b.hiveCloser = hive

	kad, err := kademlia.New( aiscAddress, addressbook, hive, p2ps, logger,
		kademlia.Options{Bootnodes: bootnodes, BootnodeMode: o.BootnodeMode, StaticNodes: o.StaticNodes, DataDir: o.DataDir})
	if err != nil {
		return nil, fmt.Errorf("unable to create kademlia: %w", err)
	}
	b.topologyCloser = kad
	b.topologyHalter = kad
	hive.SetAddPeersHandler(kad.AddPeers)
	p2ps.SetPickyNotifier(kad)

	paymentThreshold, _ := new(big.Int).SetString(o.PaymentThreshold, 10)
	lightPaymentThreshold := new(big.Int).Div(paymentThreshold, big.NewInt(lightFactor))

	pricer := pricer.NewFixedPricer( aiscAddress, basePrice)

	pricing := pricing.New(p2ps, logger, paymentThreshold, lightPaymentThreshold, big.NewInt(minPaymentThreshold))
	if err = p2ps.AddProtocol(pricing.Protocol()); err != nil {
		return nil, fmt.Errorf("pricing service: %w", err)
	}

	acc, err := accounting.NewAccounting(
		paymentThreshold,
		o.PaymentTolerance,
		o.PaymentEarly,
		logger,
		stateStore,
		pricing,
		big.NewInt(refreshRate),
		lightFactor,
		p2ps,
	)
	if err != nil {
		return nil, fmt.Errorf("accounting: %w", err)
	}
	b.accountingCloser = acc

	// bootstraper mode uses the light node refresh rate
	enforcedRefreshRate := big.NewInt(lightRefreshRate)

	pseudosettleService := pseudosettle.New(p2ps, logger, stateStore, acc, enforcedRefreshRate, enforcedRefreshRate, p2ps)
	if err = p2ps.AddProtocol(pseudosettleService.Protocol()); err != nil {
		return nil, fmt.Errorf("pseudosettle service: %w", err)
	}

	acc.SetRefreshFunc(pseudosettleService.Pay)

	pricing.SetPaymentThresholdObserver(acc)

	localStore, err := storer.New(ctx, "", &storer.Options{
		CacheCapacity: 1_000_000,
	})
	if err != nil {
		return nil, fmt.Errorf("local store creation: %w", err)
	}
	b.localstoreCloser = localStore

	radiusF := func() (uint8, error) { return  aisc.MaxBins, nil }

	retrieve := retrieval.New( aiscAddress, radiusF, localStore, p2ps, kad, logger, acc, pricer, tracer, o.RetrievalCaching)
	if err = p2ps.AddProtocol(retrieve.Protocol()); err != nil {
		return nil, fmt.Errorf("retrieval service: %w", err)
	}
	b.retrievalCloser = retrieve

	localStore.SetRetrievalService(retrieve)

	if err := kad.Start(p2pCtx); err != nil {
		return nil, err
	}

	if err := p2ps.Ready(); err != nil {
		return nil, err
	}

	if err := waitPeers(kad); err != nil {
		return nil, errors.New("timed out waiting for kademlia peers")
	}

	logger.Info("bootstrap: trying to fetch stamps snapshot")

	var (
		snapshotReference  aisc.Address
		reader            file.Joiner
		l                 int64
		eventsJSON        []byte
	)

	for i := 0; i < getSnapshotRetries; i++ {
		if err != nil {
			time.Sleep(retryWait)
		}

		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		snapshotReference, err = getLatestSnapshot(ctx, localStore.Download(true), snapshotFeed)
		if err != nil {
			logger.Warning("bootstrap: fetching snapshot failed", "error", err)
			continue
		}
		break
	}
	if err != nil {
		return nil, err
	}

	for i := 0; i < getSnapshotRetries; i++ {
		if err != nil {
			time.Sleep(retryWait)
		}

		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		reader, l, err = joiner.New(ctx, localStore.Download(true), localStore.Cache(), snapshotReference)
		if err != nil {
			logger.Warning("bootstrap: file joiner failed", "error", err)
			continue
		}

		eventsJSON, err = io.ReadAll(reader)
		if err != nil {
			logger.Warning("bootstrap: reading failed", "error", err)
			continue
		}

		if len(eventsJSON) != int(l) {
			err = errDataMismatch
			logger.Warning("bootstrap: count mismatch", "error", err)
			continue
		}
		break
	}
	if err != nil {
		return nil, err
	}

	events := postage.ChainSnapshot{}
	err = json.Unmarshal(eventsJSON, &events)
	if err != nil {
		return nil, err
	}

	return &events, nil
}

// wait till some peers are connected. returns true if all is ok
func waitPeers(kad *kademlia.Kad) error {
	const minPeersCount = 25
	return spinlock.WaitWithInterval(time.Minute, time.Second, func() bool {
		count := 0
		_ = kad.EachConnectedPeer(func(_  aisc.Address, _ uint8) (bool, bool, error) {
			count++
			return false, false, nil
		}, topology.Select{})
		return count >= minPeersCount
	})
}

func getLatestSnapshot(
	ctx context.Context,
	st storage.Getter,
	address  aisc.Address,
) ( aisc.Address, error) {
	ls := loadsave.NewReadonly(st)
	feedFactory := factory.New(st)

	m, err := manifest.NewDefaultManifestReference(
		address,
		ls,
	)
	if err != nil {
		return  aisc.ZeroAddress, fmt.Errorf("not a manifest: %w", err)
	}

	e, err := m.Lookup(ctx, "/")
	if err != nil {
		return  aisc.ZeroAddress, fmt.Errorf("node lookup: %w", err)
	}

	var (
		owner, topic []byte
		t            = new(feeds.Type)
	)
	meta := e.Metadata()
	if e := meta[" aisc-feed-owner"]; e != "" {
		owner, err = hex.DecodeString(e)
		if err != nil {
			return  aisc.ZeroAddress, err
		}
	}
	if e := meta[" aisc-feed-topic"]; e != "" {
		topic, err = hex.DecodeString(e)
		if err != nil {
			return  aisc.ZeroAddress, err
		}
	}
	if e := meta[" aisc-feed-type"]; e != "" {
		err := t.FromString(e)
		if err != nil {
			return  aisc.ZeroAddress, err
		}
	}
	if len(owner) == 0 || len(topic) == 0 {
		return  aisc.ZeroAddress, fmt.Errorf("node lookup: %s", "feed metadata absent")
	}
	f := feeds.New(topic, common.BytesToAddress(owner))

	l, err := feedFactory.NewLookup(*t, f)
	if err != nil {
		return  aisc.ZeroAddress, fmt.Errorf("feed lookup failed: %w", err)
	}

	u, _, _, err := l.At(ctx, time.Now().Unix(), 0)
	if err != nil {
		return  aisc.ZeroAddress, err
	}

	_, ref, err := feeds.FromChunk(u)
	if err != nil {
		return  aisc.ZeroAddress, err
	}

	return  aisc.NewAddress(ref), nil
}

func batchStoreExists(s storage.StateStorer) (bool, error) {

	hasOne := false
	err := s.Iterate("batchstore_", func(key, value []byte) (stop bool, err error) {
		hasOne = true
		return true, err
	})

	return hasOne, err
}
