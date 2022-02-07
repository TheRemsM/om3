package lsnrraw

import (
	"net"
	"os"
	"strings"
)

func (t *T) stop() error {
	if err := (*t.listener).Close(); err != nil {
		t.log.Error().Err(err).Msg("close failed")
		return err
	}
	t.log.Info().Msg("listener stopped")
	return nil
}

func (t *T) start() error {
	if err := os.RemoveAll(t.addr); err != nil {
		t.log.Error().Err(err).Msg("RemoveAll")
		return err
	}
	listener, err := net.Listen("unix", t.addr)
	if err != nil {
		t.log.Error().Err(err).Msg("listen failed")
		return err
	}
	c := make(chan bool)
	go func() {
		c <- true
		for {
			conn, err := listener.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					break
				} else {
					t.log.Error().Err(err).Msg("Accept")
					continue
				}
			}
			go t.mux.Serve(conn)
		}
	}()
	t.listener = &listener
	<-c
	t.log.Info().Msg("listener started " + t.addr)
	return nil
}
