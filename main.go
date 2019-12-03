package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/libp2p/go-libp2p/config"
	"github.com/libp2p/go-tcp-transport"
)

var (
	log = logging.Logger("main")
)

func main() {
	logging.SetLogLevel("*", "info")

	debug := flag.Bool("debug", false, "debug logs")
	quicFlag := flag.Bool("quic", false, "enables quic as a prefered transport for pingpong")
	flag.Parse()

	if *debug {
		logging.SetLogLevel("*", "debug")
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := createAndRunHost(ctx, *quicFlag); err != nil {
		log.Errorf("error when creating & running host: %v", err)
		os.Exit(1)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Info("Graceful shutdown...")
	cancel()
	// No waiting here...
}

func createAndRunHost(ctx context.Context, preferQUIC bool) error {
	// Support TCP by default
	opts := []config.Option{
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"), // Bind to any port available
	}
	// Want QUIC transport preference?
	if preferQUIC {
		opts = append(opts, libp2p.Transport(quic.NewTransport))
		opts = append(opts, libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/quic"))
	}
	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return err
	}
	log.Infof("My ID is %s, quic_preference = %v", h.ID(), preferQUIC)

	RegisterSayPreferedAddr(h, preferQUIC)
	RegisterPingPong(h)

	// Setup Mdns for peer-discovery on local network
	return setupMdns(ctx, h)
}
