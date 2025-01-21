package baseerr

import (
	"errors"

	"github.com/ngicks/go-common/serr"
)

func WrapBase() error {
	if Opt != nil {
		return serr.WithStackOpt(ErrBase, Opt)
	} else {
		return serr.WithStack(ErrBase)
	}
}

// define anything under this line. Adding lines above WrapBase changes stack info.

var ErrBase = errors.New("base")

var (
	DefaultOpt = &serr.WrapStackOpt{
		Override: false,
		Depth:    -1,
		Skip:     3,
	}
	Opt = &serr.WrapStackOpt{
		Override: false,
	}
)
