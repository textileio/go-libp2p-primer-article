package main

import (
	"context"
	"encoding/json"
	"io/ioutil"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	protoSayMyAddr = "/saymyaddrs/1.0.0"
)

type sayMyAddr struct {
	host          host.Host
	quicPreferred bool
}

func RegisterSayPreferedAddr(h host.Host, preferQUIC bool) {
	sma := &sayMyAddr{host: h, quicPreferred: preferQUIC}
	h.SetStreamHandler(protoSayMyAddr, sma.Handler)
}

// If a new Stream is opened, we immediately return all our prefered addrs
// to be dialed
func (h *sayMyAddr) Handler(s network.Stream) {
	defer s.Close()
	var preferedProtocol = multiaddr.P_TCP
	if h.quicPreferred {
		preferedProtocol = multiaddr.P_QUIC
	}
	var preferedAddrs []multiaddr.Multiaddr
	for _, a := range h.host.Addrs() {
		if _, err := a.ValueForProtocol(preferedProtocol); err == nil {
			preferedAddrs = append(preferedAddrs, a)
		}
	}
	addrsBytes, err := json.Marshal(&preferedAddrs)
	if err != nil {
		s.Reset()
		log.Errorf("error when marshaling my addrs: %v", err)
		return
	}
	if _, err = s.Write(addrsBytes); err != nil {
		s.Reset()
		log.Errorf("error when sending marshaled addrs: %v", err)
		return
	}
}

func askPeerForPreferredAddrs(h host.Host, id peer.ID) []multiaddr.Multiaddr {
	// Leverage this addr to discover more using `SayMyAddr` protocol
	s, err := h.NewStream(context.Background(), id, protoSayMyAddr)
	if err != nil {
		log.Errorf("error when creating stream with discovered peer: %v", err)
		return nil
	}
	defer s.Close()

	// Receive the slice of stringified multiaddrs from the discovered peer
	marshaledAddrs, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		log.Errorf("error when reading from stream: %v", err)
		return nil
	}
	var peerAddrs []string
	if err = json.Unmarshal(marshaledAddrs, &peerAddrs); err != nil {
		s.Reset()
		log.Errorf("error when unmarshaling addrs: %v", err)
		return nil
	}

	res := make([]multiaddr.Multiaddr, len(peerAddrs))
	for i := range peerAddrs {
		ma, _ := multiaddr.NewMultiaddr(peerAddrs[i])
		res[i] = ma
	}
	return res
}
