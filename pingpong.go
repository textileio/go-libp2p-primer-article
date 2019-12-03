package main

import (
	"context"
	"io/ioutil"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	protoPingPong = "/pingpong/1.0.0"
)

type pingPong struct {
	host host.Host
}

func RegisterPingPong(h host.Host) {
	pp := &pingPong{host: h}
	// Here is where we register our _pingpong_ protocol.
	// In the future when you add features/fixes to your protocol
	// you can make the current one backwards compatible, or
	// you'll need to register a new handler with the new major
	// version. If you want, you can use semver logic too, see
	// here: http://bit.ly/2YaJsJr
	h.SetStreamHandler(protoPingPong, pp.Handler)
}

// This is our handler for pingpong streams. As a parameter
// we receive a Stream, which is a bidirectional-reliable stream
// of bytes that implements io.ReadWriteCloser
func (pp *pingPong) Handler(s network.Stream) {
	defer s.Close()
	conn := s.Conn()
	log.Infof("<-- Pong to %s", conn.RemoteMultiaddr())
	if _, err := s.Write([]byte{0}); err != nil {
		s.Reset() // Notify the other side we can't continue anymore
		log.Error("error when writing pong message: %v", err)
		return
	}
}

func playPingPong(h host.Host, id peer.ID) {
	s, err := h.NewStream(context.Background(), id, protoPingPong)
	if err != nil {
		log.Errorf("error when creating stream to play pingpong: %v", err)
		return
	}
	defer s.Close()
	log.Infof("--> Ping: to %s", s.Conn().RemoteMultiaddr())
	_, err = ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		log.Errorf("error when reading from pingpong stream: %v", err)
		return
	}
	// Is important to read the stream until EOF to not leak the stream.
	// In our case is true by ioutil.ReadAll().
}
