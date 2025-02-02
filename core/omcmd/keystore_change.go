package commands

import (
	"context"

	"github.com/opensvc/om3/core/naming"
	"github.com/opensvc/om3/core/object"
	"github.com/opensvc/om3/core/objectaction"
)

type (
	CmdKeystoreChange struct {
		OptsGlobal
		Key   string
		From  string
		Value string
	}
)

func (t *CmdKeystoreChange) Run(selector, kind string) error {
	mergedSelector := mergeSelector(selector, t.ObjectSelector, kind, "")
	return objectaction.New(
		objectaction.LocalFirst(),
		objectaction.WithLocal(t.Local),
		objectaction.WithColor(t.Color),
		objectaction.WithOutput(t.Output),
		objectaction.WithObjectSelector(mergedSelector),
		objectaction.WithLocalFunc(func(ctx context.Context, p naming.Path) (interface{}, error) {
			store, err := object.NewKeystore(p)
			if err != nil {
				return nil, err
			}
			switch {
			case t.From != "":
				return nil, store.ChangeKeyFrom(t.Key, t.From)
			default:
				return nil, store.ChangeKey(t.Key, []byte(t.Value))
			}

		}),
	).Do()
}
