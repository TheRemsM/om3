package hb

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"opensvc.com/opensvc/core/clusterhb"
	"opensvc.com/opensvc/core/hbcfg"
	"opensvc.com/opensvc/core/hbtype"
	"opensvc.com/opensvc/core/kind"
	"opensvc.com/opensvc/core/path"
	"opensvc.com/opensvc/daemon/daemonctx"
	"opensvc.com/opensvc/daemon/daemondata"
	"opensvc.com/opensvc/daemon/hb/hbctrl"
	"opensvc.com/opensvc/daemon/hbcache"
	"opensvc.com/opensvc/daemon/msgbus"
	"opensvc.com/opensvc/daemon/routinehelper"
	"opensvc.com/opensvc/daemon/subdaemon"
	"opensvc.com/opensvc/util/funcopt"
	"opensvc.com/opensvc/util/hostname"
	"opensvc.com/opensvc/util/pubsub"
)

type (
	T struct {
		*subdaemon.T
		routinehelper.TT
		log          zerolog.Logger
		routineTrace routineTracer
		rootDaemon   subdaemon.RootManager
		txs          map[string]hbtype.Transmitter
		rxs          map[string]hbtype.Receiver

		ctrlC        chan<- any
		readMsgQueue chan *hbtype.Msg

		registerTxC   chan registerTxQueue
		unregisterTxC chan string

		ridSignature map[string]string
	}

	registerTxQueue struct {
		id string
		// msgToSendQueue is the queue on which a tx fetch messages to send
		msgToSendQueue chan []byte
	}

	routineTracer interface {
		Trace(string) func()
		Stats() routinehelper.Stat
	}
)

func New(opts ...funcopt.O) *T {
	t := &T{}
	t.log = log.Logger.With().Str("sub", "hb").Logger()
	t.SetTracer(routinehelper.NewTracerNoop())
	if err := funcopt.Apply(t, opts...); err != nil {
		t.log.Error().Err(err).Msg("hb funcopt.Apply")
		return nil
	}
	t.T = subdaemon.New(
		subdaemon.WithName("hb"),
		subdaemon.WithMainManager(t),
		subdaemon.WithRoutineTracer(&t.TT),
	)
	t.txs = make(map[string]hbtype.Transmitter)
	t.rxs = make(map[string]hbtype.Receiver)
	t.readMsgQueue = make(chan *hbtype.Msg)
	t.ridSignature = make(map[string]string)
	return t
}

// MainStart starts heartbeat components
//
// It starts:
// - the hb controller to maintain heartbeat status and peers
// - the dispatcher of messages to send to hb tx components
// - the dispatcher of read messages from hb rx components to daemon data
// - the launcher of tx, rx components found in configuration
func (t *T) MainStart(ctx context.Context) error {
	t.ctrlC = hbctrl.Start(ctx)

	err := t.msgToTx(ctx)
	if err != nil {
		return err
	}

	go t.msgFromRx(ctx)

	err = t.janitorHb(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (t *T) MainStop() error {
	hbToStop := make([]hbtype.IdStopper, 0)
	var failedIds []string
	for _, hb := range t.txs {
		hbToStop = append(hbToStop, hb)
	}
	for _, hb := range t.rxs {
		hbToStop = append(hbToStop, hb)
	}
	for _, hb := range hbToStop {
		if err := t.stopHb(hb); err != nil {
			t.log.Error().Err(err).Msgf("failure during stop %s", hb.Id())
			failedIds = append(failedIds, hb.Id())
		}
	}
	if len(failedIds) > 0 {
		return errors.New("failure while stopping heartbeat " + strings.Join(failedIds, ", "))
	}
	return nil
}

func (t *T) stopHb(hb hbtype.IdStopper) error {
	hbId := hb.Id()
	switch hb.(type) {
	case hbtype.Transmitter:
		t.unregisterTxC <- hbId
	}
	t.ctrlC <- hbctrl.CmdUnregister{Id: hbId}
	return hb.Stop()
}

func (t *T) startHb(hb hbcfg.Confer) error {
	if err := t.startHbRx(hb); err != nil {
		return err
	}

	if err := t.startHbTx(hb); err != nil {
		return err
	}
	return nil
}

func (t *T) startHbTx(hb hbcfg.Confer) error {
	tx := hb.Tx()
	if tx == nil {
		return errors.New("nil tx for " + hb.Name())
	}
	t.ctrlC <- hbctrl.CmdRegister{Id: tx.Id()}
	localDataC := make(chan []byte)
	if err := tx.Start(t.ctrlC, localDataC); err != nil {
		t.log.Error().Err(err).Msgf("starting %s", tx.Id())
		t.ctrlC <- hbctrl.CmdSetState{Id: tx.Id(), State: "failed"}
		return err
	}
	t.registerTxC <- registerTxQueue{id: tx.Id(), msgToSendQueue: localDataC}
	t.txs[hb.Name()] = tx
	return nil
}

func (t *T) startHbRx(hb hbcfg.Confer) error {
	rx := hb.Rx()
	if rx == nil {
		return errors.New("nil rx for " + hb.Name())
	}
	t.ctrlC <- hbctrl.CmdRegister{Id: rx.Id()}
	if err := rx.Start(t.ctrlC, t.readMsgQueue); err != nil {
		t.ctrlC <- hbctrl.CmdSetState{Id: rx.Id(), State: "failed"}
		t.log.Error().Err(err).Msgf("starting %s", rx.Id())
		return err
	}
	t.rxs[hb.Name()] = rx
	return nil
}

func (t *T) stopHbRid(rid string) error {
	errCount := 0
	failures := make([]string, 0)
	if tx, ok := t.txs[rid]; ok {
		if err := t.stopHb(tx); err != nil {
			failures = append(failures, "tx")
			errCount++
		} else {
			delete(t.txs, rid)
		}
	}
	if rx, ok := t.rxs[rid]; ok {
		if err := t.stopHb(rx); err != nil {
			failures = append(failures, "rx")
			errCount++
		} else {
			delete(t.rxs, rid)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("stop hb rid %s error for " + strings.Join(failures, ", "))
	}
	return nil
}

// rescanHb updates the running heartbeats from existing configuration
//
// To avoid hold resources, the updates are done in this order:
// 1- stop the running heartbeats that don't anymore exist in configuration
// 2- stop the running heartbeats where configuration has been changed
// 3- start the configuration changed stopped heartbeats
// 4- start the new configuration heartbeats
func (t *T) rescanHb(ctx context.Context) error {
	errs := make([]string, 0)
	ridHb, err := t.getHbConfigured(ctx)
	if err != nil {
		return err
	}
	ridSignatureNew := make(map[string]string)
	for rid, hb := range ridHb {
		ridSignatureNew[rid] = hb.Signature()
	}

	for rid := range t.ridSignature {
		if _, ok := ridSignatureNew[rid]; ok {
			continue
		}
		t.log.Info().Msgf("heartbeat config deleted %s => stopping", rid)
		if err := t.stopHbRid(rid); err == nil {
			delete(t.ridSignature, rid)
		} else {
			errs = append(errs, err.Error())
		}
	}
	// Stop first to release connexion holders
	stoppedRids := make(map[string]string)
	for rid, newSig := range ridSignatureNew {
		if sig, ok := t.ridSignature[rid]; ok {
			if sig != newSig {
				t.log.Info().Msgf("heartbeat config changed %s => stopping", rid)
				if err := t.stopHbRid(rid); err != nil {
					errs = append(errs, err.Error())
					continue
				}
				stoppedRids[rid] = newSig
			}
		}
	}
	for rid, newSig := range stoppedRids {
		t.log.Info().Msgf("heartbeat config changed %s => starting (from stoppped)", rid)
		if err := t.startHb(ridHb[rid]); err != nil {
			errs = append(errs, err.Error())
		}
		t.ridSignature[rid] = newSig
	}
	for rid, newSig := range ridSignatureNew {
		if _, ok := t.ridSignature[rid]; !ok {
			t.log.Info().Msgf("heartbeat config new %s => starting", rid)
			if err := t.startHb(ridHb[rid]); err != nil {
				errs = append(errs, err.Error())
				continue
			}
		}
		t.ridSignature[rid] = newSig
	}
	if len(errs) > 0 {
		return fmt.Errorf("rescanHb errors: %s", errs)
	}
	return nil
}

// msgToTx starts a msg multiplexer data messages to hb tx drivers
func (t *T) msgToTx(ctx context.Context) error {
	dataC := daemonctx.HBSendQ(ctx)
	if dataC == nil {
		return errors.New("msgToTx unable to retrieve HBSendQ")
	}
	t.registerTxC = make(chan registerTxQueue)
	t.unregisterTxC = make(chan string)
	go func() {
		registeredTxMsgQueue := make(map[string]chan []byte)
		for {
			select {
			case <-ctx.Done():
				return
			case c := <-t.registerTxC:
				t.log.Debug().Msgf("add %s to hb transmitters", c.id)
				registeredTxMsgQueue[c.id] = c.msgToSendQueue
			case txId := <-t.unregisterTxC:
				t.log.Debug().Msgf("remove %s from hb transmitters", txId)
				delete(registeredTxMsgQueue, txId)
			case d := <-dataC:
				for _, txQueue := range registeredTxMsgQueue {
					txQueue <- d
				}
			}
		}
	}()
	return nil
}

func (t *T) msgFromRx(ctx context.Context) {
	count := 0.0
	statTicker := time.NewTicker(10 * time.Second)
	defer statTicker.Stop()
	daemonData := daemondata.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-statTicker.C:
			t.log.Debug().Msgf("received message: %.2f/s, goroutines %d", count/10, runtime.NumGoroutine())
			count = 0
		case msg := <-t.readMsgQueue:
			t.log.Debug().Msgf("received msg type %s from %s gens: %v", msg.Kind, msg.Nodename, msg.Gen)
			switch msg.Kind {
			case "patch":
				mode := fmt.Sprintf("%d", len(msg.Deltas))
				hbcache.SetFromPeerMsg(msg.Nodename, mode, msg.Gen)
				err := daemonData.ApplyPatch(msg.Nodename, msg)
				if err != nil {
					t.log.Error().Err(err).Msgf("ApplyPatch %s from %s gens: %v", msg.Kind, msg.Nodename, msg.Gen)
				}
			case "full":
				hbcache.SetFromPeerMsg(msg.Nodename, msg.Kind, msg.Gen)
				daemonData.ApplyFull(msg.Nodename, &msg.Full)
				daemonData.ApplyFull(msg.Nodename, &msg.Full)
			case "ping":
				hbcache.SetFromPeerMsg(msg.Nodename, msg.Kind, msg.Gen)
				daemonData.ApplyPing(msg.Nodename)
			}
			count++
		}
	}
}

func (t *T) janitorHb(ctx context.Context) error {
	bus := pubsub.BusFromContext(ctx)
	clusterPath := path.T{Name: "cluster", Kind: kind.Ccfg}
	janitorC := make(chan *msgbus.Msg)
	janitorCb := func(i any) {
		janitorC <- msgbus.NewMsg(i)
	}
	msgbus.SubCfg(bus, pubsub.OpUpdate, "janitor cluster Cfg update", clusterPath.String(), janitorCb)
	msgbus.SubDaemonCtl(bus, pubsub.OpUpdate, "janitor component component", "", janitorCb)
	errC := make(chan error)

	go func(errC chan<- error) {
		errC <- t.rescanHb(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case i := <-janitorC:
				switch msg := (*i).(type) {
				case msgbus.CfgUpdated:
					if msg.Node != hostname.Hostname() {
						continue
					}
					t.log.Info().Msg("rescan heartbeat configurations (local cluster config changed)")
					if err := t.rescanHb(ctx); err != nil {
						t.log.Error().Err(err).Msg("rescan after cluster config changed")
					}
					t.log.Info().Msg("rescan heartbeat configurations done")
				case msgbus.DaemonCtl:
					hbId := msg.Component
					action := msg.Action
					if !strings.HasPrefix(hbId, "hb#") {
						continue
					}
					switch msg.Action {
					case "stop":
						t.daemonCtlStop(hbId, action)
					case "start":
						t.daemonCtlStart(ctx, hbId, action)
					}
				}
			}
		}
	}(errC)
	return <-errC
}

func (t *T) daemonCtlStart(ctx context.Context, hbId string, action string) {
	var rid string
	if strings.HasSuffix(hbId, ".rx") {
		rid = strings.TrimSuffix(hbId, ".rx")
	} else if strings.HasSuffix(hbId, ".tx") {
		rid = strings.TrimSuffix(hbId, ".tx")
	} else {
		t.log.Info().Msgf("daemonctl %s %s found no component for %s", action, hbId)
		return
	}
	h, err := t.getHbConfiguredComponent(ctx, rid)
	if err != nil {
		t.log.Info().Msgf("daemonctl %s %s found no component for %s (rid: %s)", action, hbId, rid)
		return
	}
	if strings.HasSuffix(hbId, ".rx") {
		if err := t.startHbRx(h); err != nil {
			t.log.Error().Err(err).Msgf("daemonctl %s %s failure", action, hbId)
			return
		}
	} else {
		if err := t.startHbTx(h); err != nil {
			t.log.Error().Err(err).Msgf("daemonctl %s %s failure", action, hbId)
			return
		}
	}
}

func (t *T) daemonCtlStop(hbId string, action string) {
	var hbI interface{}
	var found bool
	if strings.HasSuffix(hbId, ".rx") {
		rid := strings.TrimSuffix(hbId, ".rx")
		if hbI, found = t.rxs[rid]; !found {
			t.log.Info().Msgf("daemonctl %s %s found no %s.rx component", action, hbId, rid)
			return
		}
	} else if strings.HasSuffix(hbId, ".tx") {
		rid := strings.TrimSuffix(hbId, ".tx")
		if hbI, found = t.txs[rid]; !found {
			t.log.Info().Msgf("daemonctl %s %s found no %s.tx component", action, hbId, rid)
			return
		}
	} else {
		t.log.Info().Msgf("daemonctl %s %s found no component", action, hbId)
		return
	}
	t.log.Info().Msgf("ask to %s %s", action, hbId)
	switch hbI.(type) {
	case hbtype.Transmitter:
		t.unregisterTxC <- hbId
	}
	if err := hbI.(hbtype.IdStopper).Stop(); err != nil {
		t.log.Error().Err(err).Msgf("daemonctl %s %s failure", action, hbId)
	} else {
		t.ctrlC <- hbctrl.CmdSetState{Id: hbI.(hbtype.IdStopper).Id(), State: "stopped"}
	}
}

func (t *T) getHbConfigured(ctx context.Context) (ridHb map[string]hbcfg.Confer, err error) {
	var node *clusterhb.T
	ridHb = make(map[string]hbcfg.Confer)
	node, err = clusterhb.New()
	if err != nil {
		return ridHb, err
	}
	for _, h := range node.Hbs() {
		h.Configure(ctx)
		ridHb[h.Name()] = h
	}
	return ridHb, nil
}

func (t *T) getHbConfiguredComponent(ctx context.Context, rid string) (c hbcfg.Confer, err error) {
	var node *clusterhb.T
	node, err = clusterhb.New()
	if err != nil {
		t.log.Error().Err(err).Msgf("clusterhb.New")
		return
	}
	for _, h := range node.Hbs() {
		h.Configure(ctx)
		if h.Name() == rid {
			c = h
			return
		}
	}
	err = errors.New("not found rid")
	return
}
