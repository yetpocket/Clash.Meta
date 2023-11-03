package control

import (
	"errors"
	"syscall"

	"github.com/sagernet/sing/common/atomic"

	"golang.org/x/sys/unix"
)

var ifIndexDisabled atomic.Bool

func bindToInterface(conn syscall.RawConn, network string, address string, interfaceName string, interfaceIndex int) error {
	return Raw(conn, func(fd uintptr) error {
		if !ifIndexDisabled.Load() {
			err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_BINDTOIFINDEX, interfaceIndex)
			if err == nil {
				return nil
			} else if errors.Is(err, unix.ENOPROTOOPT) {
				ifIndexDisabled.Store(true)
			} else {
				return err
			}
		}
		return unix.BindToDevice(int(fd), interfaceName)
	})
}
