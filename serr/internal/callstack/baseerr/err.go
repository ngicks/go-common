package baseerr

import (
	"errors"

	"github.com/ngicks/go-common/serr"
)

var ErrBase = errors.New("base")

var (
	Override = false
	Depth    = -1
)

func WrapBase() error {
	if Depth >= 0 {
		return serr.WithStackOverride(ErrBase, Override, Depth)
	} else {
		return serr.WithStack(ErrBase)
	}
}
