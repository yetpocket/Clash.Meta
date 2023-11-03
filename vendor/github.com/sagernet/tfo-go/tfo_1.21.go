//go:build go1.21

package tfo

import "errors"

func (PlatformUnsupportedError) Is(target error) bool {
	return target == errors.ErrUnsupported
}
