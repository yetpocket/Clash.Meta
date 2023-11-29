package dialer

import (
	"context"
	"net"
	"net/netip"
	"syscall"

	"github.com/Dreamacro/clash/component/iface"
	"github.com/Dreamacro/clash/log"

	"golang.org/x/sys/unix"
)

func bindControl(ifaceIdx int) controlFn {
	return func(ctx context.Context, network, address string, c syscall.RawConn) (err error) {
		ifaceName, err := net.InterfaceByIndex(ifaceIdx)
		if err != nil {
			log.Debugln("darwin bind %s resolve interface %d error at %+v", address, ifaceIdx, err)
		} else {
			log.Debugln("darwin bind %s to interface %s", address, ifaceName.Name)
		}
		addrPort, err := netip.ParseAddrPort(address)
		if err == nil && !addrPort.Addr().IsGlobalUnicast() {
			return
		}

		var innerErr error
		err = c.Control(func(fd uintptr) {
			switch network {
			case "tcp4", "udp4":
				innerErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, ifaceIdx)
			case "tcp6", "udp6":
				innerErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, ifaceIdx)
			}
		})

		if innerErr != nil {
			err = innerErr
		}

		return
	}
}

func bindIfaceToDialer(ifaceName string, dialer *net.Dialer, _ string, _ netip.Addr) error {
	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return err
	}
	log.Debugln("bind to interface %s", ifaceName)
	addControlToDialer(dialer, bindControl(ifaceObj.Index))
	return nil
}

func bindIfaceToListenConfig(ifaceName string, lc *net.ListenConfig, _, address string) (string, error) {
	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return "", err
	}

	addControlToListenConfig(lc, bindControl(ifaceObj.Index))
	return address, nil
}

func ParseNetwork(network string, addr netip.Addr) string {
	return network
}
