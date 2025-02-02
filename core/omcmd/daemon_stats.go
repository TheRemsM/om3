package commands

import (
	"encoding/json"
	"fmt"

	"github.com/opensvc/om3/core/cluster"
)

type (
	CmdDaemonStats struct {
		OptsGlobal
	}
)

func (t *CmdDaemonStats) Run() error {
	/*
		var (
			err  error
			b    []byte
			data cluster.Stats
			c    *client.T
		)
		c, err = client.New(client.WithURL(t.Server))
		if err != nil {
			return err
		}
		resp, err := c.GetDaemonStats(context.Background())
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		data, err = parseDaemonStats(b)
		if err != nil {
			return err
		}
		renderer := output.Renderer{
			Format:   t.Format,
			Color:    t.Color,
			Data:     data,
			Colorize: rawconfig.Colorize,
		}
		renderer.Print()
	*/
	return fmt.Errorf("todo")
}

func parseDaemonStats(b []byte) (cluster.Stats, error) {
	type (
		nodeData struct {
			Status int                     `json:"status"`
			Data   cluster.NodeStatsBundle `json:"data"`
		}
		responseType struct {
			Status int                 `json:"status"`
			Nodes  map[string]nodeData `json:"nodes"`
		}
	)
	var t responseType
	ds := make(cluster.Stats)
	err := json.Unmarshal(b, &t)
	if err != nil {
		return ds, err
	}
	for k, v := range t.Nodes {
		ds[k] = v.Data
	}
	return ds, nil
}
