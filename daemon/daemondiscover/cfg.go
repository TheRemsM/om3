package daemondiscover

import (
	"context"
	"os"
	"time"

	"opensvc.com/opensvc/core/instance"
	"opensvc.com/opensvc/core/kind"
	"opensvc.com/opensvc/core/path"
	"opensvc.com/opensvc/core/rawconfig"
	"opensvc.com/opensvc/daemon/daemonctx"
	ps "opensvc.com/opensvc/daemon/daemonps"
	"opensvc.com/opensvc/daemon/monitor/instcfg"
	"opensvc.com/opensvc/daemon/monitor/moncmd"
	"opensvc.com/opensvc/daemon/remoteconfig"
	"opensvc.com/opensvc/util/file"
	"opensvc.com/opensvc/util/pubsub"
	"opensvc.com/opensvc/util/timestamp"
)

func (d *discover) cfg() {
	d.log.Info().Msg("cfg started")
	defer func() {
		done := time.After(dropCmdTimeout)
		for {
			select {
			case <-done:
				return
			case <-d.cfgCmdC:
			}
		}
	}()
	c := daemonctx.DaemonPubSubCmd(d.ctx)
	defer ps.UnSub(c, ps.SubCfg(c, pubsub.OpUpdate, "discover.cfg cfg.update", "", d.onEvCfg))
	defer ps.UnSub(c, ps.SubCfg(c, pubsub.OpDelete, "discover.cfg cfg.delete", "", d.onEvCfg))

	for {
		select {
		case <-d.ctx.Done():
			d.log.Info().Msg("cfg done")
			return
		case i := <-d.cfgCmdC:
			switch c := (*i).(type) {
			case moncmd.CfgFsWatcherCreate:
				d.cmdLocalCfgFileAdded(c.Path, c.Filename)
			case moncmd.MonCfgDone:
				d.cmdInstCfgDone(c.Path, c.Filename)
			case moncmd.CfgUpdated:
				if c.Node == d.localhost {
					continue
				}
				d.cmdRemoteCfgUpdated(c.Path, c.Node, c.Config)
			case moncmd.CfgDeleted:
				if c.Node == d.localhost {
					continue
				}
				d.cmdRemoteCfgDeleted(c.Path, c.Node)
			case moncmd.RemoteFileConfig:
				d.cmdRemoteCfgFetched(c)
			default:
				d.log.Error().Interface("cmd", i).Msg("unknown cmd")
			}
		}
	}
}

func (d *discover) onEvCfg(i interface{}) {
	d.cfgCmdC <- moncmd.New(i)
}

func (d *discover) cmdLocalCfgFileAdded(p path.T, filename string) {
	s := p.String()
	if _, ok := d.moncfg[s]; ok {
		return
	}
	instcfg.Start(d.ctx, p, filename, d.cfgCmdC)
	d.moncfg[s] = struct{}{}
}

func (d *discover) cmdInstCfgDone(p path.T, filename string) {
	s := p.String()
	if _, ok := d.moncfg[s]; ok {
		delete(d.moncfg, s)
	}
	if file.Exists(filename) {
		d.cmdLocalCfgFileAdded(p, filename)
	}
}

func (d *discover) cmdRemoteCfgUpdated(p path.T, node string, remoteCfg instance.Config) {
	s := p.String()
	if _, ok := d.moncfg[s]; ok {
		return
	}
	d.log.Info().Msgf("cmdRemoteCfgUpdated for node %s, path %s", node, p)
	if remoteUpdated, ok := d.fetcherUpdated[s]; ok {
		// fetcher in progress for s
		if remoteCfg.Updated.Time().After(remoteUpdated.Time()) {
			d.log.Info().Msgf("cancel pending remote cfg fetcher, more recent config from %s on %s", s, node)
			d.cancelFetcher(s)
		} else {
			d.log.Error().Msgf("cmdRemoteCfgUpdated for node %s, path %s not more recent", node, p)
			return
		}
	}
	if p.Kind != kind.Sec && !d.inScope(&remoteCfg) {
		d.log.Error().Msgf("cmdRemoteCfgUpdated for node %s, path %s not in scope", node, p)
		return
	}
	d.log.Info().Msgf("fetch config %s from node %s", s, node)
	d.fetchCfgFromRemote(p, node, remoteCfg.Updated)
}

func (d *discover) cmdRemoteCfgDeleted(p path.T, node string) {
	s := p.String()
	if fetchFrom, ok := d.fetcherFrom[s]; ok {
		if fetchFrom == node {
			d.log.Info().Msgf("cancel pending remote cfg fetcher %s@%s not anymore present", s, node)
			d.cancelFetcher(s)
		}
	}
}

func (d *discover) cmdRemoteCfgFetched(c moncmd.RemoteFileConfig) {
	select {
	case <-c.Ctx.Done():
		c.Err <- nil
		return
	default:
		defer d.cancelFetcher(c.Path.String())
		var prefix string
		if c.Path.Namespace != "root" {
			prefix = "namespaces/"
		}
		s := c.Path.String()
		confFile := rawconfig.Paths.Etc + "/" + prefix + s + ".conf"
		d.log.Info().Msgf("install fetched config %s from %s", s, c.Node)
		err := os.Rename(c.Filename, confFile)
		if err != nil {
			d.log.Error().Err(err).Msgf("can't install fetched config to %s", confFile)
		}
		c.Err <- err
	}
	return
}

func (d *discover) inScope(cfg *instance.Config) bool {
	localhost := d.localhost
	for _, node := range cfg.Scope {
		if node == localhost {
			return true
		}
	}
	return false
}

func (d *discover) cancelFetcher(s string) {
	node := d.fetcherFrom[s]
	d.fetcherCancel[s]()
	delete(d.fetcherCancel, s)
	delete(d.fetcherNodeCancel[node], s)
	delete(d.fetcherUpdated, s)
	delete(d.fetcherFrom, s)
}

func (d *discover) fetchCfgFromRemote(p path.T, node string, updated timestamp.T) {
	s := p.String()
	if n, ok := d.fetcherFrom[s]; ok {
		d.log.Error().Msgf("fetcher already in progress for %s from %s", s, n)
		return
	}
	ctx, cancel := context.WithCancel(d.ctx)
	d.fetcherCancel[s] = cancel
	d.fetcherFrom[s] = node
	d.fetcherUpdated[s] = updated
	if _, ok := d.fetcherNodeCancel[node]; ok {
		d.fetcherNodeCancel[node][s] = cancel
	} else {
		d.fetcherNodeCancel[node] = make(map[string]context.CancelFunc)
	}

	go remoteconfig.Fetch(ctx, p, node, d.cfgCmdC)
}
