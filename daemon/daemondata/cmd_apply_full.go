package daemondata

import (
	"encoding/json"

	"opensvc.com/opensvc/core/cluster"
	"opensvc.com/opensvc/core/event"
	"opensvc.com/opensvc/daemon/daemonps"
	"opensvc.com/opensvc/util/jsondelta"
	"opensvc.com/opensvc/util/timestamp"
)

type opApplyRemoteFull struct {
	nodename string
	full     *cluster.NodeStatus
	done     chan<- bool
}

func (o opApplyRemoteFull) call(d *data) {
	d.counterCmd <- idApplyFull
	d.log.Debug().Msgf("opApplyRemoteFull %s", o.nodename)
	d.pending.Monitor.Nodes[o.nodename] = *o.full
	d.mergedFromPeer[o.nodename] = o.full.Gen[o.nodename]
	d.remotesNeedFull[o.nodename] = false
	if gen, ok := d.pending.Monitor.Nodes[o.nodename].Gen[d.localNode]; ok {
		d.mergedOnPeer[o.nodename] = gen
	}

	absolutePatch := jsondelta.Patch{
		jsondelta.Operation{
			OpPath:  jsondelta.OperationPath{"monitor", "nodes", o.nodename},
			OpValue: jsondelta.NewOptValue(o.full),
			OpKind:  "replace",
		},
	}

	if eventB, err := json.Marshal(absolutePatch); err != nil {
		d.log.Error().Err(err).Msgf("Marshal absolutePatch %s", o.nodename)
	} else {
		var eventData json.RawMessage = eventB
		eventId++
		daemonps.PubEvent(d.bus, event.Event{
			Kind:      "patch",
			ID:        eventId,
			Timestamp: timestamp.Now(),
			Data:      &eventData,
		})
	}

	d.log.Debug().
		Interface("remotesNeedFull", d.remotesNeedFull).
		Interface("mergedOnPeer", d.mergedOnPeer).
		Interface("pending gen", d.pending.Monitor.Nodes[o.nodename].Gen).
		Interface("full.gen", o.full.Gen).
		Msgf("opApplyRemoteFull %s", o.nodename)
	o.done <- true
}

func (t T) ApplyFull(nodename string, full *cluster.NodeStatus) {
	done := make(chan bool)
	t.cmdC <- opApplyRemoteFull{
		nodename: nodename,
		full:     full,
		done:     done,
	}
	<-done
}
