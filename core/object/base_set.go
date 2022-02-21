package object

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"opensvc.com/opensvc/core/keyop"
	"opensvc.com/opensvc/core/xconfig"
)

// OptsSet is the options of the Set object method.
type OptsSet struct {
	Global     OptsGlobal
	Lock       OptsLocking
	KeywordOps []string `flag:"kwops"`
}

// Set changes or adds a keyword and its value in the configuration file.
func (t *Base) Set(options OptsSet) error {
	return t.lockedAction("", options.Lock, "set", func() error {
		return setKeywords(t.config, options.KeywordOps)
	})
}

func (t *Base) SetKeywords(kws []string) error {
	return setKeywords(t.config, kws)
}

func (t *Base) SetKeys(kops ...keyop.T) error {
	return setKeys(t.config, kops...)
}

func setKeywords(cf *xconfig.T, kws []string) error {
	l := make([]keyop.T, len(kws))
	for i, kw := range kws {
		op := keyop.Parse(kw)
		l[i] = *op
	}
	return setKeys(cf, l...)
}

func setKeys(cf *xconfig.T, kops ...keyop.T) error {
	changes := 0
	for _, op := range kops {
		if op.IsZero() {
			return fmt.Errorf("invalid set expression: %s", op)
		}
		log.Debug().
			Stringer("key", op.Key).
			Stringer("op", op.Op).
			Str("val", op.Value).
			Msg("set")
		if err := cf.Set(op); err != nil {
			return err
		}
		changes++
	}
	if changes > 0 {
		return cf.Commit()
	}
	return nil
}
