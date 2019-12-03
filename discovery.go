package main

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery"
)

func setupMdns(ctx context.Context, h host.Host) error {
	// We're going to be announcing every 5 seconds
	mdns, err := discovery.NewMdnsService(ctx, h, time.Second*5, "")
	if err != nil {
		return err
	}
	notifee := &MdnsNotifee{
		h:          h,
		askedPeers: make(map[peer.ID]struct{}),
	}
	mdns.RegisterNotifee(notifee)
	return nil
}

type MdnsNotifee struct {
	lock       sync.Mutex
	h          host.Host
	askedPeers map[peer.ID]struct{}
}

// When a new peer is found, we run a new Stream of `SayMyAddrs` protocol where
// we receive his/her preferred multiaddrs to be dialed.
func (mh *MdnsNotifee) HandlePeerFound(ai peer.AddrInfo) {
	mh.lock.Lock()
	defer mh.lock.Unlock()

	// Already Ping/Pong peer?
	if _, ok := mh.askedPeers[ai.ID]; ok {
		return
	}
	mh.askedPeers[ai.ID] = struct{}{}
	log.Infof("discovered unknown peer: %s", ai.ID)

	// We add the discovered addrs from Mdns to the peer known addrs
	mh.h.Peerstore().AddAddrs(ai.ID, ai.Addrs, peerstore.PermanentAddrTTL)
	addrs := askPeerForPreferredAddrs(mh.h, ai.ID)
	// We define that our discovered peer addrs are what they prefer
	mh.h.Peerstore().ClearAddrs(ai.ID)
	mh.h.Peerstore().SetAddrs(ai.ID, addrs, peerstore.PermanentAddrTTL)
	// Close existing connection with the peer, so new Streams will dial from
	// preferred addresses
	if err := mh.h.Network().ClosePeer(ai.ID); err != nil {
		log.Warning("error when closing existing connection with peer")
	}

	go playPingPong(mh.h, ai.ID)
}
