// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kademlia

import (
	"context"
	random "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/aisc/pkg/addressbook"
	"github.com/aisc/pkg/discovery"
	"github.com/aisc/pkg/log"
	"github.com/aisc/pkg/p2p"
	"github.com/aisc/pkg/shed"
	"github.com/aisc/pkg/ aisc"
	"github.com/aisc/pkg/topology"
	im "github.com/aisc/pkg/topology/kademlia/internal/metrics"
	"github.com/aisc/pkg/topology/kademlia/internal/waitnext"
	"github.com/aisc/pkg/topology/pslice"
	ma "github.com/multiformats/go-multiaddr"
	"golang.org/x/sync/errgroup"
)

// loggerName is the tree path name of the logger for this package.
const loggerName = "kademlia"

const (
	maxConnAttempts     = 1 // when there is maxConnAttempts failed connect calls for a given peer it is considered non-connectable
	maxBootNodeAttempts = 3 // how many attempts to dial to boot-nodes before giving up
	maxNeighborAttempts = 3 // how many attempts to dial to boot-nodes before giving up

	addPeerBatchSize = 500

	// To avoid context.Timeout errors during network failure, the value of
	// the peerConnectionAttemptTimeout constant must be equal to or greater
	// than 5 seconds (empirically verified).
	peerConnectionAttemptTimeout = 15 * time.Second // timeout for establishing a new connection with peer.
)

// Default option values
const (
	defaultBitSuffixLength             = 4 // the number of bits used to create pseudo addresses for balancing, 2^4, 16 addresses
	defaultLowWaterMark                = 3 // the number of peers in consecutive deepest bins that constitute as nearest neighbours
	defaultSaturationPeers             = 8
	defaultOverSaturationPeers         = 20
	defaultBootNodeOverSaturationPeers = 20
	defaultShortRetry                  = 30 * time.Second
	defaultTimeToRetry                 = 2 * defaultShortRetry
	defaultBroadcastBinSize            = 4
)

var (
	errOverlayMismatch   = errors.New("overlay mismatch")
	errPruneEntry        = errors.New("prune entry")
	errEmptyBin          = errors.New("empty bin")
	errAnnounceLightNode = errors.New("announcing light node")
)

type (
	binSaturationFunc  func(bin uint8, peers, connected *pslice.PSlice, filter peerFilterFunc) bool
	sanctionedPeerFunc func(peer  aisc.Address) bool
	pruneFunc          func(depth uint8)
	staticPeerFunc     func(peer  aisc.Address) bool
	peerFilterFunc     func(peer  aisc.Address) bool
	filtersFunc        func(...im.FilterOp) peerFilterFunc
)

var noopSanctionedPeerFn = func(_  aisc.Address) bool { return false }

// Options for injecting services to Kademlia.
type Options struct {
	SaturationFunc binSaturationFunc
	Bootnodes      []ma.Multiaddr
	BootnodeMode   bool
	PruneFunc      pruneFunc
	StaticNodes    [] aisc.Address
	FilterFunc     filtersFunc
	DataDir        string

	BitSuffixLength             *int
	TimeToRetry                 *time.Duration
	ShortRetry                  *time.Duration
	SaturationPeers             *int
	OverSaturationPeers         *int
	BootnodeOverSaturationPeers *int
	BroadcastBinSize            *int
	LowWaterMark                *int
}

// kadOptions are made from Options with default values set
type kadOptions struct {
	SaturationFunc binSaturationFunc
	Bootnodes      []ma.Multiaddr
	BootnodeMode   bool
	PruneFunc      pruneFunc
	StaticNodes    [] aisc.Address
	FilterFunc     filtersFunc

	TimeToRetry                 time.Duration
	ShortRetry                  time.Duration
	BitSuffixLength             int // additional depth of common prefix for bin
	SaturationPeers             int
	OverSaturationPeers         int
	BootnodeOverSaturationPeers int
	BroadcastBinSize            int
	LowWaterMark                int
}

func newKadOptions(o Options) kadOptions {
	ko := kadOptions{
		// copy values
		SaturationFunc: o.SaturationFunc,
		Bootnodes:      o.Bootnodes,
		BootnodeMode:   o.BootnodeMode,
		PruneFunc:      o.PruneFunc,
		StaticNodes:    o.StaticNodes,
		FilterFunc:     o.FilterFunc,
		// copy or use default
		TimeToRetry:                 defaultValDuration(o.TimeToRetry, defaultTimeToRetry),
		ShortRetry:                  defaultValDuration(o.ShortRetry, defaultShortRetry),
		BitSuffixLength:             defaultValInt(o.BitSuffixLength, defaultBitSuffixLength),
		SaturationPeers:             defaultValInt(o.SaturationPeers, defaultSaturationPeers),
		OverSaturationPeers:         defaultValInt(o.OverSaturationPeers, defaultOverSaturationPeers),
		BootnodeOverSaturationPeers: defaultValInt(o.BootnodeOverSaturationPeers, defaultBootNodeOverSaturationPeers),
		BroadcastBinSize:            defaultValInt(o.BroadcastBinSize, defaultBroadcastBinSize),
		LowWaterMark:                defaultValInt(o.LowWaterMark, defaultLowWaterMark),
	}

	if ko.SaturationFunc == nil {
		ko.SaturationFunc = makeSaturationFunc(ko)
	}

	return ko
}

func defaultValInt(v *int, d int) int {
	if v == nil {
		return d
	}
	return *v
}

func defaultValDuration(v *time.Duration, d time.Duration) time.Duration {
	if v == nil {
		return d
	}
	return *v
}

func makeSaturationFunc(o kadOptions) binSaturationFunc {
	os := o.OverSaturationPeers
	if o.BootnodeMode {
		os = o.BootnodeOverSaturationPeers
	}
	return binSaturated(os, isStaticPeer(o.StaticNodes))
}

// Kad is the Aisc forwarding kademlia implementation.
type Kad struct {
	opt               kadOptions
	base               aisc.Address         // this node's overlay address
	discovery         discovery.Driver      // the discovery driver
	addressBook       addressbook.Interface // address book to get underlays
	p2p               p2p.Service           // p2p service to connect to nodes with
	commonBinPrefixes [][] aisc.Address     // list of address prefixes for each bin
	connectedPeers    *pslice.PSlice        // a slice of peers sorted and indexed by po, indexes kept in `bins`
	knownPeers        *pslice.PSlice        // both are po aware slice of addresses
	depth             uint8                 // current neighborhood depth
	storageRadius     uint8                 // storage area of responsibility
	depthMu           sync.RWMutex          // protect depth changes
	manageC           chan struct{}         // trigger the manage forever loop to connect to new peers
	peerSig           []chan struct{}
	peerSigMtx        sync.Mutex
	logger            log.Logger // logger
	bootnode          bool       // indicates whether the node is working in bootnode mode
	collector         *im.Collector
	quit              chan struct{} // quit channel
	halt              chan struct{} // halt channel
	done              chan struct{} // signal that `manage` has quit
	wg                sync.WaitGroup
	waitNext          *waitnext.WaitNext
	metrics           metrics
	staticPeer        staticPeerFunc
	bgBroadcastCtx    context.Context
	bgBroadcastCancel context.CancelFunc
	reachability      p2p.ReachabilityStatus
}

// New returns a new Kademlia.
func New(
	base  aisc.Address,
	addressbook addressbook.Interface,
	discovery discovery.Driver,
	p2pSvc p2p.Service,
	logger log.Logger,
	o Options,
) (*Kad, error) {
	var k *Kad

	if o.DataDir == "" {
		logger.Warning("using in-mem store for kademlia metrics, no state will be persisted")
	} else {
		o.DataDir = filepath.Join(o.DataDir, "kademlia-metrics")
	}
	sdb, err := shed.NewDB(o.DataDir, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create metrics storage: %w", err)
	}
	imc, err := im.NewCollector(sdb)
	if err != nil {
		return nil, fmt.Errorf("unable to create metrics collector: %w", err)
	}

	opt := newKadOptions(o)

	k = &Kad{
		opt:               opt,
		base:              base,
		discovery:         discovery,
		addressBook:       addressbook,
		p2p:               p2pSvc,
		commonBinPrefixes: make([][] aisc.Address, int( aisc.MaxBins)),
		connectedPeers:    pslice.New(int( aisc.MaxBins), base),
		knownPeers:        pslice.New(int( aisc.MaxBins), base),
		manageC:           make(chan struct{}, 1),
		waitNext:          waitnext.New(),
		logger:            logger.WithName(loggerName).Register(),
		bootnode:          opt.BootnodeMode,
		collector:         imc,
		quit:              make(chan struct{}),
		halt:              make(chan struct{}),
		done:              make(chan struct{}),
		metrics:           newMetrics(),
		staticPeer:        isStaticPeer(opt.StaticNodes),
		storageRadius:      aisc.MaxPO,
	}

	if k.opt.PruneFunc == nil {
		k.opt.PruneFunc = k.pruneOversaturatedBins
	}

	if k.opt.FilterFunc == nil {
		k.opt.FilterFunc = func(f ...im.FilterOp) peerFilterFunc {
			return func(peer  aisc.Address) bool {
				return k.collector.Filter(peer, f...)
			}
		}
	}

	if k.opt.BitSuffixLength > 0 {
		k.commonBinPrefixes = generateCommonBinPrefixes(k.base, k.opt.BitSuffixLength)
	}

	k.bgBroadcastCtx, k.bgBroadcastCancel = context.WithCancel(context.Background())

	k.metrics.ReachabilityStatus.WithLabelValues(p2p.ReachabilityStatusUnknown.String()).Set(0)
	return k, nil
}

type peerConnInfo struct {
	po   uint8
	addr  aisc.Address
}

// connectBalanced attempts to connect to the balanced peers first.
func (k *Kad) connectBalanced(wg *sync.WaitGroup, peerConnChan chan<- *peerConnInfo) {
	skipPeers := func(peer  aisc.Address) bool {
		if k.waitNext.Waiting(peer) {
			k.metrics.TotalBeforeExpireWaits.Inc()
			return true
		}
		return false
	}

	depth := k.neighborhoodDepth()

	for i := range k.commonBinPrefixes {

		binPeersLength := k.knownPeers.BinSize(uint8(i))

		// balancer should skip on bins where neighborhood connector would connect to peers anyway
		// and there are not enough peers in known addresses to properly balance the bin
		if i >= int(depth) && binPeersLength < len(k.commonBinPrefixes[i]) {
			continue
		}

		binPeers := k.knownPeers.BinPeers(uint8(i))
		binConnectedPeers := k.connectedPeers.BinPeers(uint8(i))

		for j := range k.commonBinPrefixes[i] {
			pseudoAddr := k.commonBinPrefixes[i][j]

			// Connect to closest known peer which we haven't tried connecting to recently.

			_, exists := nClosePeerInSlice(binConnectedPeers, pseudoAddr, noopSanctionedPeerFn, uint8(i+k.opt.BitSuffixLength+1))
			if exists {
				continue
			}

			closestKnownPeer, exists := nClosePeerInSlice(binPeers, pseudoAddr, skipPeers, uint8(i+k.opt.BitSuffixLength+1))
			if !exists {
				continue
			}

			if k.connectedPeers.Exists(closestKnownPeer) {
				continue
			}

			blocklisted, err := k.p2p.Blocklisted(closestKnownPeer)
			if err != nil {
				k.logger.Warning("peer blocklist check failed", "error", err)
			}
			if blocklisted {
				continue
			}

			wg.Add(1)
			select {
			case peerConnChan <- &peerConnInfo{
				po:    aisc.Proximity(k.base.Bytes(), closestKnownPeer.Bytes()),
				addr: closestKnownPeer,
			}:
			case <-k.quit:
				wg.Done()
				return
			}
		}
	}
}

// connectNeighbours attempts to connect to the neighbours
// which were not considered by the connectBalanced method.
func (k *Kad) connectNeighbours(wg *sync.WaitGroup, peerConnChan chan<- *peerConnInfo) {

	sent := 0
	var currentPo uint8 = 0

	_ = k.knownPeers.EachBinRev(func(addr  aisc.Address, po uint8) (bool, bool, error) {

		// out of depth, skip bin
		if po < k.neighborhoodDepth() {
			return false, true, nil
		}

		if po != currentPo {
			currentPo = po
			sent = 0
		}

		if k.connectedPeers.Exists(addr) {
			return false, false, nil
		}

		blocklisted, err := k.p2p.Blocklisted(addr)
		if err != nil {
			k.logger.Warning("peer blocklist check failed", "error", err)
		}
		if blocklisted {
			return false, false, nil
		}

		if k.waitNext.Waiting(addr) {
			k.metrics.TotalBeforeExpireWaits.Inc()
			return false, false, nil
		}

		wg.Add(1)
		select {
		case peerConnChan <- &peerConnInfo{po: po, addr: addr}:
		case <-k.quit:
			wg.Done()
			return true, false, nil
		}

		sent++

		// We want 'sent' equal to 'saturationPeers'
		// in order to skip to the next bin and speed up the topology build.
		return false, sent == k.opt.SaturationPeers, nil
	})
}

// connectionAttemptsHandler handles the connection attempts
// to peers sent by the producers to the peerConnChan.
func (k *Kad) connectionAttemptsHandler(ctx context.Context, wg *sync.WaitGroup, neighbourhoodChan, balanceChan <-chan *peerConnInfo) {
	connect := func(peer *peerConnInfo) {
		aiscAddr, err := k.addressBook.Get(peer.addr)
		switch {
		case errors.Is(err, addressbook.ErrNotFound):
			k.logger.Debug("empty address book entry for peer", "peer_address", peer.addr)
			k.knownPeers.Remove(peer.addr)
			return
		case err != nil:
			k.logger.Debug("failed to get address book entry for peer", "peer_address", peer.addr, "error", err)
			return
		}

		remove := func(peer *peerConnInfo) {
			k.waitNext.Remove(peer.addr)
			k.knownPeers.Remove(peer.addr)
			if err := k.addressBook.Remove(peer.addr); err != nil {
				k.logger.Debug("could not remove peer from addressbook", "peer_address", peer.addr)
			}
		}

		switch err = k.connect(ctx, peer.addr, aiscAddr.Underlay); {
		case errors.Is(err, p2p.ErrNetworkUnavailable):
			k.logger.Debug("network unavailable when reaching peer", "peer_overlay_address", peer.addr, "peer_underlay_address", aiscAddr.Underlay)
			return
		case errors.Is(err, errPruneEntry):
			k.logger.Debug("dial to light node", "peer_overlay_address", peer.addr, "peer_underlay_address", aiscAddr.Underlay)
			remove(peer)
			return
		case errors.Is(err, errOverlayMismatch):
			k.logger.Debug("overlay mismatch has occurred", "peer_overlay_address", peer.addr, "peer_underlay_address", aiscAddr.Underlay)
			remove(peer)
			return
		case errors.Is(err, p2p.ErrPeerBlocklisted):
			k.logger.Debug("peer still in blocklist", "peer_address", aiscAddr)
			k.logger.Warning("peer still in blocklist")
			return
		case err != nil:
			k.logger.Debug("peer not reachable from kademlia", "peer_address", aiscAddr, "error", err)
			k.logger.Warning("peer not reachable when attempting to connect")
			return
		}

		k.waitNext.Set(peer.addr, time.Now().Add(k.opt.ShortRetry), 0)

		k.connectedPeers.Add(peer.addr)

		k.metrics.TotalOutboundConnections.Inc()
		k.collector.Record(peer.addr, im.PeerLogIn(time.Now(), im.PeerConnectionDirectionOutbound))

		k.recalcDepth()

		k.logger.Info("connected to peer", "peer_address", peer.addr, "proximity_order", peer.po)
		k.notifyManageLoop()
		k.notifyPeerSig()
	}

	var (
		// The inProgress helps to avoid making a connection
		// to a peer who has the connection already in progress.
		inProgress   = make(map[string]bool)
		inProgressMu sync.Mutex
	)
	connAttempt := func(peerConnChan <-chan *peerConnInfo) {
		for {
			select {
			case <-k.quit:
				return
			case peer := <-peerConnChan:
				addr := peer.addr.String()

				if k.waitNext.Waiting(peer.addr) {
					k.metrics.TotalBeforeExpireWaits.Inc()
					wg.Done()
					continue
				}

				inProgressMu.Lock()
				if !inProgress[addr] {
					inProgress[addr] = true
					inProgressMu.Unlock()
					connect(peer)
					inProgressMu.Lock()
					delete(inProgress, addr)
				}
				inProgressMu.Unlock()
				wg.Done()
			}
		}
	}
	for i := 0; i < 32; i++ {
		go connAttempt(balanceChan)
	}
	for i := 0; i < 32; i++ {
		go connAttempt(neighbourhoodChan)
	}
}

// notifyManageLoop notifies kademlia manage loop.
func (k *Kad) notifyManageLoop() {
	select {
	case k.manageC <- struct{}{}:
	default:
	}
}

// manage is a forever loop that manages the connection to new peers
// once they get added or once others leave.
func (k *Kad) manage() {
	loggerV1 := k.logger.V(1).Register()

	defer k.wg.Done()
	defer close(k.done)
	defer k.logger.Debug("kademlia manage loop exited")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-k.quit
		cancel()
	}()

	// The wg makes sure that we wait for all the connection attempts,
	// spun up by goroutines, to finish before we try the boot-nodes.
	var wg sync.WaitGroup
	neighbourhoodChan := make(chan *peerConnInfo)
	balanceChan := make(chan *peerConnInfo)
	go k.connectionAttemptsHandler(ctx, &wg, neighbourhoodChan, balanceChan)

	k.wg.Add(1)
	go func() {
		defer k.wg.Done()
		for {
			select {
			case <-k.halt:
				return
			case <-k.quit:
				return
			case <-time.After(5 * time.Minute):
				start := time.Now()
				loggerV1.Debug("starting to flush metrics", "start_time", start)
				if err := k.collector.Flush(); err != nil {
					k.metrics.InternalMetricsFlushTotalErrors.Inc()
					k.logger.Debug("unable to flush metrics counters to the persistent store", "error", err)
				} else {
					k.metrics.InternalMetricsFlushTime.Observe(time.Since(start).Seconds())
					loggerV1.Debug("flush metrics done", "elapsed", time.Since(start))
				}
			}
		}
	}()

	// tell each neighbor about other neighbors periodically
	k.wg.Add(1)
	go func() {
		defer k.wg.Done()
		for {
			select {
			case <-k.halt:
				return
			case <-k.quit:
				return
			case <-time.After(5 * time.Minute):
				var neighbors [] aisc.Address
				_ = k.connectedPeers.EachBin(func(addr  aisc.Address, bin uint8) (stop bool, jumpToNext bool, err error) {
					if bin < k.neighborhoodDepth() {
						return true, false, nil
					}
					neighbors = append(neighbors, addr)
					return false, false, nil
				})
				for i, peer := range neighbors {
					if err := k.discovery.BroadcastPeers(ctx, peer, append(neighbors[:i], neighbors[i+1:]...)...); err != nil {
						k.logger.Debug("broadcast neighborhood failure", "peer_address", peer, "error", err)
					}
				}
			}
		}
	}()

	for {
		select {
		case <-k.quit:
			return
		case <-time.After(15 * time.Second):
			k.notifyManageLoop()
		case <-k.manageC:
			start := time.Now()

			select {
			case <-k.halt:
				// halt stops dial-outs while shutting down
				return
			case <-k.quit:
				return
			default:
			}

			if k.bootnode {
				depth := k.neighborhoodDepth()

				k.metrics.CurrentDepth.Set(float64(depth))
				k.metrics.CurrentlyKnownPeers.Set(float64(k.knownPeers.Length()))
				k.metrics.CurrentlyConnectedPeers.Set(float64(k.connectedPeers.Length()))

				continue
			}

			oldDepth := k.neighborhoodDepth()
			k.connectBalanced(&wg, balanceChan)
			k.connectNeighbours(&wg, neighbourhoodChan)
			wg.Wait()

			depth := k.neighborhoodDepth()

			k.opt.PruneFunc(depth)

			loggerV1.Debug("connector finished", "elapsed", time.Since(start), "old_depth", oldDepth, "new_depth", depth)

			k.metrics.CurrentDepth.Set(float64(depth))
			k.metrics.CurrentlyKnownPeers.Set(float64(k.knownPeers.Length()))
			k.metrics.CurrentlyConnectedPeers.Set(float64(k.connectedPeers.Length()))

			if k.connectedPeers.Length() == 0 {
				select {
				case <-k.halt:
					continue
				default:
				}
				k.logger.Debug("kademlia: no connected peers, trying bootnodes")
				k.connectBootNodes(ctx)
			} else {
				rs := make(map[string]float64)
				ss := k.collector.Snapshot(time.Now())

				if err := k.connectedPeers.EachBin(func(addr  aisc.Address, _ uint8) (bool, bool, error) {
					if ss, ok := ss[addr.ByteString()]; ok {
						rs[ss.Reachability.String()]++
					}
					return false, false, nil
				}); err != nil {
					k.logger.Error(err, "unable to set peers reachability status")
				}

				for status, count := range rs {
					k.metrics.PeersReachabilityStatus.WithLabelValues(status).Set(count)
				}
			}
		}
	}
}

// pruneOversaturatedBins disconnects out of depth peers from oversaturated bins
// while maintaining the balance of the bin and favoring healthy and reachable peers.
func (k *Kad) pruneOversaturatedBins(depth uint8) {

	for i := range k.commonBinPrefixes {

		if i >= int(depth) {
			return
		}

		binPeersCount := k.connectedPeers.BinSize(uint8(i))
		if binPeersCount <= k.opt.OverSaturationPeers {
			continue
		}

		for j := 0; j < len(k.commonBinPrefixes[i]) && k.connectedPeers.BinSize(uint8(i)) > k.opt.OverSaturationPeers; j++ {

			binPeers := k.connectedPeers.BinPeers(uint8(i))
			peers := k.balancedSlotPeers(k.commonBinPrefixes[i][j], binPeers, i)
			if len(peers) <= 1 {
				continue
			}

			var disconnectPeer =  aisc.ZeroAddress
			var unreachablePeer =  aisc.ZeroAddress
			for _, peer := range peers {
				if ss := k.collector.Inspect(peer); ss != nil {
					if !ss.Healthy {
						disconnectPeer = peer
						break
					}
					if ss.Reachability != p2p.ReachabilityStatusPublic {
						unreachablePeer = peer
					}
				}
			}

			if disconnectPeer.IsZero() {
				if unreachablePeer.IsZero() {
					disconnectPeer = peers[rand.Intn(len(peers))]
				} else {
					disconnectPeer = unreachablePeer // pick unrechable peer
				}
			}

			err := k.p2p.Disconnect(disconnectPeer, "pruned from oversaturated bin")
			if err != nil {
				k.logger.Debug("prune disconnect failed", "error", err)
			}
		}

		k.logger.Debug("pruning", "bin", i, "oldBinSize", binPeersCount, "newBinSize", k.connectedPeers.BinSize(uint8(i)))
	}
}

func (k *Kad) balancedSlotPeers(pseudoAddr  aisc.Address, peers [] aisc.Address, po int) [] aisc.Address {

	var ret [] aisc.Address

	for _, peer := range peers {
		peerPo :=  aisc.ExtendedProximity(peer.Bytes(), pseudoAddr.Bytes())
		if int(peerPo) >= po+k.opt.BitSuffixLength+1 {
			ret = append(ret, peer)
		}
	}

	return ret
}

func (k *Kad) Start(_ context.Context) error {
	k.wg.Add(1)
	go k.manage()

	k.AddPeers(k.previouslyConnected()...)

	go func() {
		select {
		case <-k.halt:
			return
		case <-k.quit:
			return
		default:
		}
		var (
			start     = time.Now()
			addresses [] aisc.Address
		)

		err := k.addressBook.IterateOverlays(func(addr  aisc.Address) (stop bool, err error) {
			addresses = append(addresses, addr)
			if len(addresses) == addPeerBatchSize {
				k.AddPeers(addresses...)
				addresses = nil
			}
			return false, nil
		})
		if err != nil {
			k.logger.Error(err, "addressbook iterate overlays failed")
			return
		}
		k.AddPeers(addresses...)
		k.metrics.StartAddAddressBookOverlaysTime.Observe(time.Since(start).Seconds())
	}()

	// trigger the first manage loop immediately so that
	// we can start connecting to the bootnode quickly
	k.notifyManageLoop()

	return nil
}

func (k *Kad) previouslyConnected() [] aisc.Address {
	loggerV1 := k.logger.V(1).Register()

	now := time.Now()
	ss := k.collector.Snapshot(now)
	loggerV1.Debug("metrics snapshot taken", "elapsed", time.Since(now))

	var peers [] aisc.Address

	for addr, p := range ss {
		if p.ConnectionTotalDuration > 0 {
			peers = append(peers,  aisc.NewAddress([]byte(addr)))
		}
	}

	return peers
}

func (k *Kad) connectBootNodes(ctx context.Context) {
	loggerV1 := k.logger.V(1).Register()

	var attempts, connected int
	totalAttempts := maxBootNodeAttempts * len(k.opt.Bootnodes)

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for _, addr := range k.opt.Bootnodes {
		if attempts >= totalAttempts || connected >= 3 {
			return
		}

		if _, err := p2p.Discover(ctx, addr, func(addr ma.Multiaddr) (stop bool, err error) {
			loggerV1.Debug("connecting to bootnode", "bootnode_address", addr)
			if attempts >= maxBootNodeAttempts {
				return true, nil
			}
			aiscAddress, err := k.p2p.Connect(ctx, addr)

			attempts++
			k.metrics.TotalBootNodesConnectionAttempts.Inc()

			if err != nil {
				if !errors.Is(err, p2p.ErrAlreadyConnected) {
					k.logger.Debug("connect to bootnode failed", "bootnode_address", addr, "error", err)
					k.logger.Warning("connect to bootnode failed", "bootnode_address", addr)
					return false, err
				}
				k.logger.Debug("connect to bootnode failed", "bootnode_address", addr, "error", err)
				return false, nil
			}

			if err := k.onConnected(ctx, aiscAddress.Overlay); err != nil {
				return false, err
			}

			k.metrics.TotalOutboundConnections.Inc()
			k.collector.Record(aiscAddress.Overlay, im.PeerLogIn(time.Now(), im.PeerConnectionDirectionOutbound))
			loggerV1.Debug("connected to bootnode", "bootnode_address", addr)
			connected++

			// connect to max 3 bootnodes
			return connected >= 3, nil
		}); err != nil && !errors.Is(err, context.Canceled) {
			k.logger.Debug("discover to bootnode failed", "bootnode_address", addr, "error", err)
			k.logger.Warning("discover to bootnode failed", "bootnode_address", addr)
			return
		}
	}
}

// binSaturated indicates whether a certain bin is saturated or not.
// when a bin is not saturated it means we would like to proactively
// initiate connections to other peers in the bin.
func binSaturated(oversaturationAmount int, staticNode staticPeerFunc) binSaturationFunc {
	return func(bin uint8, peers, connected *pslice.PSlice, filter peerFilterFunc) bool {
		size := 0
		_ = connected.EachBin(func(addr  aisc.Address, po uint8) (bool, bool, error) {
			if po == bin && !filter(addr) && !staticNode(addr) {
				size++
			}
			return false, false, nil
		})

		return size >= oversaturationAmount
	}
}

// recalcDepth calculates, assigns the new depth, and returns if depth has changed
func (k *Kad) recalcDepth() {

	k.depthMu.Lock()
	defer k.depthMu.Unlock()

	var (
		peers                 = k.connectedPeers
		filter                = k.opt.FilterFunc(im.Reachability(false))
		binCount              = 0
		shallowestUnsaturated = uint8(0)
		depth                 uint8
	)

	// handle edge case separately
	if peers.Length() <= k.opt.LowWaterMark {
		k.depth = 0
		return
	}

	_ = peers.EachBinRev(func(addr  aisc.Address, bin uint8) (bool, bool, error) {
		if filter(addr) {
			return false, false, nil
		}
		if bin == shallowestUnsaturated {
			binCount++
			return false, false, nil
		}
		if bin > shallowestUnsaturated && binCount < k.opt.SaturationPeers {
			// this means we have less than quickSaturationPeers in the previous bin
			// therefore we can return assuming that bin is the unsaturated one.
			return true, false, nil
		}
		shallowestUnsaturated = bin
		binCount = 1

		return false, false, nil
	})
	depth = shallowestUnsaturated

	shallowestEmpty, noEmptyBins := peers.ShallowestEmpty()
	// if there are some empty bins and the shallowestEmpty is
	// smaller than the shallowestUnsaturated then set shallowest
	// unsaturated to the empty bin.
	if !noEmptyBins && shallowestEmpty < depth {
		depth = shallowestEmpty
	}

	var (
		peersCtr  = uint(0)
		candidate = uint8(0)
	)
	_ = peers.EachBin(func(addr  aisc.Address, po uint8) (bool, bool, error) {
		if filter(addr) {
			return false, false, nil
		}
		peersCtr++
		if peersCtr >= uint(k.opt.LowWaterMark) {
			candidate = po
			return true, false, nil
		}
		return false, false, nil
	})

	if depth > candidate {
		depth = candidate
	}

	k.depth = depth
}

// connect connects to a peer and gossips its address to our connected peers,
// as well as sends the peers we are connected to the newly connected peer
func (k *Kad) connect(ctx context.Context, peer  aisc.Address, ma ma.Multiaddr) error {
	k.logger.Debug("attempting connect to peer", "peer_address", peer)

	ctx, cancel := context.WithTimeout(ctx, peerConnectionAttemptTimeout)
	defer cancel()

	k.metrics.TotalOutboundConnectionAttempts.Inc()

	switch i, err := k.p2p.Connect(ctx, ma); {
	case errors.Is(err, p2p.ErrNetworkUnavailable):
		return err
	case k.p2p.NetworkStatus() == p2p.NetworkStatusUnavailable:
		return p2p.ErrNetworkUnavailable
	case errors.Is(err, p2p.ErrDialLightNode):
		return errPruneEntry
	case errors.Is(err, p2p.ErrAlreadyConnected):
		if !i.Overlay.Equal(peer) {
			return errOverlayMismatch
		}
		return nil
	case errors.Is(err, context.Canceled):
		return err
	case errors.Is(err, p2p.ErrPeerBlocklisted):
		return err
	case err != nil:
		k.logger.Debug("could not connect to peer", "peer_address", peer, "error", err)

		retryTime := time.Now().Add(k.opt.TimeToRetry)
		var e *p2p.ConnectionBackoffError
		failedAttempts := 0
		if errors.As(err, &e) {
			retryTime = e.TryAfter()
		} else {
			failedAttempts = k.waitNext.Attempts(peer)
			failedAttempts++
		}

		k.metrics.TotalOutboundConnectionFailedAttempts.Inc()
		k.collector.Record(peer, im.IncSessionConnectionRetry())

		maxAttempts := maxConnAttempts
		if  aisc.Proximity(k.base.Bytes(), peer.Bytes()) >= k.neighborhoodDepth() {
			maxAttempts = maxNeighborAttempts
		}

		if failedAttempts >= maxAttempts {
			k.waitNext.Remove(peer)
			k.knownPeers.Remove(peer)
			if err := k.addressBook.Remove(peer); err != nil {
				k.logger.Debug("could not remove peer from addressbook", "peer_address", peer)
			}
			k.logger.Debug("peer pruned from address book", "peer_address", peer)
		} else {
			k.waitNext.Set(peer, retryTime, failedAttempts)
		}

		return err
	case !i.Overlay.Equal(peer):
		_ = k.p2p.Disconnect(peer, errOverlayMismatch.Error())
		_ = k.p2p.Disconnect(i.Overlay, errOverlayMismatch.Error())
		return errOverlayMismatch
	}

	return k.Announce(ctx, peer, true)
}

// Announce a newly connected peer to our connected peers, but also
// notify the peer about our already connected peers
func (k *Kad) Announce(ctx context.Context, peer  aisc.Address, fullnode bool) error {
	var addrs [] aisc.Address

	depth := k.neighborhoodDepth()
	isNeighbor :=  aisc.Proximity(peer.Bytes(), k.base.Bytes()) >= depth

outer:
	for bin := uint8(0); bin <  aisc.MaxBins; bin++ {

		var (
			connectedPeers [] aisc.Address
			err            error
		)

		if bin >= depth && isNeighbor {
			connectedPeers = k.binPeers(bin, false) // broadcast all neighborhood peers
		} else {
			connectedPeers, err = randomSubset(k.binPeers(bin, true), k.opt.BroadcastBinSize)
			if err != nil {
				return err
			}
		}

		for _, connectedPeer := range connectedPeers {
			if connectedPeer.Equal(peer) {
				continue
			}

			addrs = append(addrs, connectedPeer)

			if !fullnode {
				// we continue here so we dont gossip
				// about lightnodes to others.
				continue
			}
			// if kademlia is closing, dont enqueue anymore broadcast requests
			select {
			case <-k.bgBroadcastCtx.Done():
				// we will not interfere with the announce operation by returning here
				continue
			case <-k.halt:
				break outer
			default:
			}
			go func(connectedPeer  aisc.Address) {

				// Create a new deadline ctx to prevent goroutine pile up
				cCtx, cCancel := context.WithTimeout(k.bgBroadcastCtx, time.Minute)
				defer cCancel()

				if err := k.discovery.BroadcastPeers(cCtx, connectedPeer, peer); err != nil {
					k.logger.Debug("peer gossip failed", "new_peer_address", peer, "connected_peer_address", connectedPeer, "error", err)
				}
			}(connectedPeer)
		}
	}

	if len(addrs) == 0 {
		return nil
	}

	select {
	case <-k.halt:
		return nil
	default:
	}

	err := k.discovery.BroadcastPeers(ctx, peer, addrs...)
	if err != nil {
		k.logger.Error(err, "could not broadcast to peer", "peer_address", peer)
		_ = k.p2p.Disconnect(peer, "failed broadcasting to peer")
	}

	return err
}

// AnnounceTo announces a selected peer to another.
func (k *Kad) AnnounceTo(ctx context.Context, addressee, peer  aisc.Address, fullnode bool) error {
	if !fullnode {
		return errAnnounceLightNode
	}

	return k.discovery.BroadcastPeers(ctx, addressee, peer)
}

// AddPeers adds peers to the knownPeers list.
// This does not guarantee that a connection will immediately
// be made to the peer.
func (k *Kad) AddPeers(addrs ... aisc.Address) {
	k.knownPeers.Add(addrs...)
	k.notifyManageLoop()
}

func (k *Kad) Pick(peer p2p.Peer) bool {
	k.metrics.PickCalls.Inc()
	if k.bootnode || !peer.FullNode {
		// shortcircuit for bootnode mode AND light node peers - always accept connections,
		// at least until we find a better solution.
		return true
	}
	po :=  aisc.Proximity(k.base.Bytes(), peer.Address.Bytes())
	oversaturated := k.opt.SaturationFunc(po, k.knownPeers, k.connectedPeers, k.opt.FilterFunc(im.Reachability(false)))
	// pick the peer if we are not oversaturated
	if !oversaturated {
		return true
	}
	k.metrics.PickCallsFalse.Inc()
	return false
}

func (k *Kad) binPeers(bin uint8, reachable bool) (peers [] aisc.Address) {

	_ = k.EachConnectedPeerRev(func(p  aisc.Address, po uint8) (bool, bool, error) {

		if po == bin {
			peers = append(peers, p)
			return false, false, nil
		}

		if po > bin {
			return true, false, nil
		}

		return false, true, nil

	}, topology.Select{Reachable: reachable})

	return
}

func isStaticPeer(staticNodes [] aisc.Address) func(overlay  aisc.Address) bool {
	return func(overlay  aisc.Address) bool {
		return  aisc.ContainsAddress(staticNodes, overlay)
	}
}

// Connected is called when a peer has dialed in.
// If forceConnection is true `overSaturated` is ignored for non-bootnodes.
func (k *Kad) Connected(ctx context.Context, peer p2p.Peer, forceConnection bool) (err error) {
	defer func() {
		if err == nil {
			k.metrics.TotalInboundConnections.Inc()
			k.collector.Record(peer.Address, im.PeerLogIn(time.Now(), im.PeerConnectionDirectionInbound))
		}
	}()

	address := peer.Address
	po :=  aisc.Proximity(k.base.Bytes(), address.Bytes())

	if overSaturated := k.opt.SaturationFunc(po, k.knownPeers, k.connectedPeers, k.opt.FilterFunc(im.Reachability(false))); overSaturated {
		if k.bootnode {
			randPeer, err := k.randomPeer(po)
			if err != nil {
				return fmt.Errorf("failed to get random peer to kick-out: %w", err)
			}
			_ = k.p2p.Disconnect(randPeer, "kicking out random peer to accommodate node")
			return k.onConnected(ctx, address)
		}
		if !forceConnection {
			return topology.ErrOversaturated
		}
	}

	return k.onConnected(ctx, address)
}

func (k *Kad) onConnected(ctx context.Context, addr  aisc.Address) error {
	if err := k.Announce(ctx, addr, true); err != nil {
		return err
	}

	k.knownPeers.Add(addr)
	k.connectedPeers.Add(addr)

	k.waitNext.Remove(addr)

	k.recalcDepth()

	k.notifyManageLoop()
	k.notifyPeerSig()

	return nil
}

// Disconnected is called when peer disconnects.
func (k *Kad) Disconnected(peer p2p.Peer) {
	k.logger.Info("disconnected peer", "peer_address", peer.Address)

	k.connectedPeers.Remove(peer.Address)

	k.waitNext.SetTryAfter(peer.Address, time.Now().Add(k.opt.TimeToRetry))

	k.metrics.TotalInboundDisconnections.Inc()
	k.collector.Record(peer.Address, im.PeerLogOut(time.Now()))

	k.recalcDepth()

	k.notifyManageLoop()
	k.notifyPeerSig()
}

func (k *Kad) notifyPeerSig() {
	k.peerSigMtx.Lock()
	defer k.peerSigMtx.Unlock()

	for _, c := range k.peerSig {
		// Every peerSig channel has a buffer capacity of 1,
		// so every receiver will get the signal even if the
		// select statement has the default case to avoid blocking.
		select {
		case c <- struct{}{}:
		default:
		}
	}
}

func nClosePeerInSlice(peers [] aisc.Address, addr  aisc.Address, spf sanctionedPeerFunc, minPO uint8) ( aisc.Address, bool) {
	for _, peer := range peers {
		if spf(peer) {
			continue
		}

		if  aisc.ExtendedProximity(peer.Bytes(), addr.Bytes()) >= minPO {
			return peer, true
		}
	}

	return  aisc.ZeroAddress, false
}

func (k *Kad) IsReachable() bool {
	return k.reachability == p2p.ReachabilityStatusPublic
}

// ClosestPeer returns the closest peer to a given address.
func (k *Kad) ClosestPeer(addr  aisc.Address, includeSelf bool, filter topology.Select, skipPeers ... aisc.Address) ( aisc.Address, error) {
	if k.connectedPeers.Length() == 0 {
		return  aisc.Address{}, topology.ErrNotFound
	}

	closest :=  aisc.ZeroAddress

	if includeSelf && k.reachability == p2p.ReachabilityStatusPublic {
		closest = k.base
	}

	err := k.EachConnectedPeerRev(func(peer  aisc.Address, po uint8) (bool, bool, error) {
		if  aisc.ContainsAddress(skipPeers, peer) {
			return false, false, nil
		}

		if closest.IsZero() {
			closest = peer
			return false, false, nil
		}

		closer, err := peer.Closer(addr, closest)
		if closer {
			closest = peer
		}
		if err != nil {
			k.logger.Debug("closest peer", "peer", peer, "addr", addr, "error", err)
		}
		return false, false, nil
	}, filter)

	if err != nil {
		return  aisc.Address{}, err
	}

	if closest.IsZero() { // no peers
		return  aisc.Address{}, topology.ErrNotFound // only for light nodes
	}

	// check if self
	if closest.Equal(k.base) {
		return  aisc.Address{}, topology.ErrWantSelf
	}

	return closest, nil
}

// EachConnectedPeer implements topology.PeerIterator interface.
func (k *Kad) EachConnectedPeer(f topology.EachPeerFunc, filter topology.Select) error {
	filters := filterOps(filter)

	return k.connectedPeers.EachBin(func(addr  aisc.Address, po uint8) (bool, bool, error) {
		if len(filters) > 0 && k.opt.FilterFunc(filters...)(addr) {
			return false, false, nil
		}
		return f(addr, po)
	})
}

// EachConnectedPeerRev implements topology.PeerIterator interface.
func (k *Kad) EachConnectedPeerRev(f topology.EachPeerFunc, filter topology.Select) error {
	filters := filterOps(filter)
	return k.connectedPeers.EachBinRev(func(addr  aisc.Address, po uint8) (bool, bool, error) {
		if len(filters) > 0 && k.opt.FilterFunc(filters...)(addr) {
			return false, false, nil
		}
		return f(addr, po)
	})
}
func (k *Kad) PeersCount(filter topology.Select) int {
	return k.connectedPeers.Length()
}

// Reachable sets the peer reachability status.
func (k *Kad) Reachable(addr  aisc.Address, status p2p.ReachabilityStatus) {
	k.collector.Record(addr, im.PeerReachability(status))
	k.logger.Debug("reachability of peer updated", "peer_address", addr, "reachability", status)
	if status == p2p.ReachabilityStatusPublic {
		k.recalcDepth()
		k.notifyManageLoop()
	}
}

// UpdateReachability updates node reachability status.
// The status will be updated only once. Updates to status
// p2p.ReachabilityStatusUnknown are ignored.
func (k *Kad) UpdateReachability(status p2p.ReachabilityStatus) {
	if status == p2p.ReachabilityStatusUnknown {
		return
	}
	k.logger.Debug("reachability updated", "reachability", status)
	k.reachability = status
	k.metrics.ReachabilityStatus.WithLabelValues(status.String()).Set(0)
}

// UpdateReachability updates node reachability status.
// The status will be updated only once. Updates to status
// p2p.ReachabilityStatusUnknown are ignored.
func (k *Kad) UpdatePeerHealth(peer  aisc.Address, health bool, dur time.Duration) {
	k.collector.Record(peer, im.PeerHealth(health), im.PeerLatency(dur))
}

// SubscribeTopologyChange returns the channel that signals when the connected peers
// set and depth changes. Returned function is safe to be called multiple times.
func (k *Kad) SubscribeTopologyChange() (c <-chan struct{}, unsubscribe func()) {
	channel := make(chan struct{}, 1)
	var closeOnce sync.Once

	k.peerSigMtx.Lock()
	defer k.peerSigMtx.Unlock()

	k.peerSig = append(k.peerSig, channel)

	unsubscribe = func() {
		k.peerSigMtx.Lock()
		defer k.peerSigMtx.Unlock()

		for i, c := range k.peerSig {
			if c == channel {
				k.peerSig = append(k.peerSig[:i], k.peerSig[i+1:]...)
				break
			}
		}

		closeOnce.Do(func() { close(channel) })
	}

	return channel, unsubscribe
}

func filterOps(filter topology.Select) []im.FilterOp {

	ops := make([]im.FilterOp, 0, 2)

	if filter.Reachable {
		ops = append(ops, im.Reachability(false))
	}
	if filter.Healthy {
		ops = append(ops, im.Health(false))
	}

	return ops
}

// NeighborhoodDepth returns the current Kademlia depth.
func (k *Kad) neighborhoodDepth() uint8 {
	k.depthMu.RLock()
	defer k.depthMu.RUnlock()

	return k.storageRadius
}

func (k *Kad) SetStorageRadius(d uint8) {

	k.depthMu.Lock()
	defer k.depthMu.Unlock()

	if k.storageRadius == d {
		return
	}

	k.storageRadius = d
	k.metrics.CurrentStorageDepth.Set(float64(k.storageRadius))
	k.logger.Debug("kademlia set storage radius", "radius", k.storageRadius)

	k.notifyManageLoop()
	k.notifyPeerSig()
}

func (k *Kad) Snapshot() *topology.KadParams {
	var infos []topology.BinInfo
	for i := int( aisc.MaxPO); i >= 0; i-- {
		infos = append(infos, topology.BinInfo{})
	}

	ss := k.collector.Snapshot(time.Now())

	_ = k.connectedPeers.EachBin(func(addr  aisc.Address, po uint8) (bool, bool, error) {
		infos[po].BinConnected++
		infos[po].ConnectedPeers = append(
			infos[po].ConnectedPeers,
			&topology.PeerInfo{
				Address: addr,
				Metrics: createMetricsSnapshotView(ss[addr.ByteString()]),
			},
		)
		return false, false, nil
	})

	// output (k.knownPeers not k.connectedPeers) here to not repeat the peers we already have in the connected peers list
	_ = k.knownPeers.EachBin(func(addr  aisc.Address, po uint8) (bool, bool, error) {
		infos[po].BinPopulation++

		for _, v := range infos[po].ConnectedPeers {
			// peer already connected, don't show in the known peers list
			if v.Address.Equal(addr) {
				return false, false, nil
			}
		}

		infos[po].DisconnectedPeers = append(
			infos[po].DisconnectedPeers,
			&topology.PeerInfo{
				Address: addr,
				Metrics: createMetricsSnapshotView(ss[addr.ByteString()]),
			},
		)
		return false, false, nil
	})

	return &topology.KadParams{
		Base:                k.base.String(),
		Population:          k.knownPeers.Length(),
		Connected:           k.connectedPeers.Length(),
		Timestamp:           time.Now(),
		NNLowWatermark:      k.opt.LowWaterMark,
		Depth:               k.neighborhoodDepth(),
		Reachability:        k.reachability.String(),
		NetworkAvailability: k.p2p.NetworkStatus().String(),
		Bins: topology.KadBins{
			Bin0:  infos[0],
			Bin1:  infos[1],
			Bin2:  infos[2],
			Bin3:  infos[3],
			Bin4:  infos[4],
			Bin5:  infos[5],
			Bin6:  infos[6],
			Bin7:  infos[7],
			Bin8:  infos[8],
			Bin9:  infos[9],
			Bin10: infos[10],
			Bin11: infos[11],
			Bin12: infos[12],
			Bin13: infos[13],
			Bin14: infos[14],
			Bin15: infos[15],
			Bin16: infos[16],
			Bin17: infos[17],
			Bin18: infos[18],
			Bin19: infos[19],
			Bin20: infos[20],
			Bin21: infos[21],
			Bin22: infos[22],
			Bin23: infos[23],
			Bin24: infos[24],
			Bin25: infos[25],
			Bin26: infos[26],
			Bin27: infos[27],
			Bin28: infos[28],
			Bin29: infos[29],
			Bin30: infos[30],
			Bin31: infos[31],
		},
	}
}

// String returns a string represenstation of Kademlia.
func (k *Kad) String() string {
	j := k.Snapshot()
	b, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		k.logger.Error(err, "could not marshal kademlia into json")
		return ""
	}
	return string(b)
}

// Halt stops outgoing connections from happening.
// This is needed while we shut down, so that further topology
// changes do not happen while we shut down.
func (k *Kad) Halt() {
	close(k.halt)
}

// Close shuts down kademlia.
func (k *Kad) Close() error {
	k.logger.Info("kademlia shutting down")
	close(k.quit)
	cc := make(chan struct{})

	k.bgBroadcastCancel()

	go func() {
		k.wg.Wait()
		close(cc)
	}()

	eg := errgroup.Group{}

	errTimeout := errors.New("timeout")

	eg.Go(func() error {
		select {
		case <-cc:
		case <-time.After(peerConnectionAttemptTimeout):
			return fmt.Errorf("kademlia shutting down with running goroutines: %w", errTimeout)
		}
		return nil
	})

	eg.Go(func() error {
		select {
		case <-k.done:
		case <-time.After(time.Second * 5):
			return fmt.Errorf("kademlia manage loop did not shut down properly: %w", errTimeout)
		}
		return nil
	})

	err := eg.Wait()

	k.logger.Info("kademlia persisting peer metrics")
	start := time.Now()
	if err := k.collector.Finalize(start, false); err != nil {
		k.logger.Debug("unable to finalize open sessions", "error", err)
	}
	k.logger.Debug("metrics collector finalized", "elapsed", time.Since(start))

	return err
}

func randomSubset(addrs [] aisc.Address, count int) ([] aisc.Address, error) {
	if count >= len(addrs) {
		return addrs, nil
	}

	for i := 0; i < len(addrs); i++ {
		b, err := random.Int(random.Reader, big.NewInt(int64(len(addrs))))
		if err != nil {
			return nil, err
		}
		j := int(b.Int64())
		addrs[i], addrs[j] = addrs[j], addrs[i]
	}

	return addrs[:count], nil
}

func (k *Kad) randomPeer(bin uint8) ( aisc.Address, error) {
	peers := k.connectedPeers.BinPeers(bin)

	for idx := 0; idx < len(peers); {
		// do not consider protected peers
		if k.staticPeer(peers[idx]) {
			peers = append(peers[:idx], peers[idx+1:]...)
			continue
		}
		idx++
	}

	if len(peers) == 0 {
		return  aisc.ZeroAddress, errEmptyBin
	}

	rndIndx, err := random.Int(random.Reader, big.NewInt(int64(len(peers))))
	if err != nil {
		return  aisc.ZeroAddress, err
	}

	return peers[rndIndx.Int64()], nil
}

// createMetricsSnapshotView creates new topology.MetricSnapshotView from the
// given metrics.Snapshot and rounds all the timestamps and durations to its
// nearest second, except for the peer latency, which is given in milliseconds.
func createMetricsSnapshotView(ss *im.Snapshot) *topology.MetricSnapshotView {
	if ss == nil {
		return nil
	}
	return &topology.MetricSnapshotView{
		LastSeenTimestamp:          time.Unix(0, ss.LastSeenTimestamp).Unix(),
		SessionConnectionRetry:     ss.SessionConnectionRetry,
		ConnectionTotalDuration:    ss.ConnectionTotalDuration.Truncate(time.Second).Seconds(),
		SessionConnectionDuration:  ss.SessionConnectionDuration.Truncate(time.Second).Seconds(),
		SessionConnectionDirection: string(ss.SessionConnectionDirection),
		LatencyEWMA:                ss.LatencyEWMA.Milliseconds(),
		Reachability:               ss.Reachability.String(),
		Healthy:                    ss.Healthy,
	}
}