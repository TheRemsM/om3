package commands

import (
	"github.com/spf13/cobra"
	"opensvc.com/opensvc/core/flag"
	"opensvc.com/opensvc/core/object"
	"opensvc.com/opensvc/core/objectaction"
	"opensvc.com/opensvc/core/path"
)

type (
	// CmdObjectRestart is the cobra flag set of the restart command.
	CmdObjectRestart struct {
		object.OptsStart
	}
)

// Init configures a cobra command and adds it to the parent command.
func (t *CmdObjectRestart) Init(kind string, parent *cobra.Command, selector *string) {
	cmd := t.cmd(kind, selector)
	parent.AddCommand(cmd)
	flag.Install(cmd, t)
}

func (t *CmdObjectRestart) cmd(kind string, selector *string) *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "restart the selected objects or resources",
		Run: func(cmd *cobra.Command, args []string) {
			t.run(selector, kind)
		},
	}
}

func (t *CmdObjectRestart) run(selector *string, kind string) {
	mergedSelector := mergeSelector(*selector, t.OptsGlobal.ObjectSelector, kind, "")
	objectaction.New(
		objectaction.WithObjectSelector(mergedSelector),
		objectaction.WithResourceSelectorOptions(t.OptsResourceSelector.Options),
		objectaction.WithLocal(t.OptsGlobal.Local),
		objectaction.WithFormat(t.OptsGlobal.Format),
		objectaction.WithColor(t.OptsGlobal.Color),
		objectaction.WithRemoteNodes(t.OptsGlobal.NodeSelector),
		objectaction.WithRemoteAction("restart"),
		objectaction.WithAsyncTarget("restarted"),
		objectaction.WithAsyncWatch(t.OptsAsync.Watch),
		objectaction.WithLocalRun(func(p path.T) (interface{}, error) {
			o, err := object.NewActorFromPath(p)
			if err != nil {
				return nil, err
			}
			return nil, o.Restart(t.OptsStart)
		}),
	).Do()
}
