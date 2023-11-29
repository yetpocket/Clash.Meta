package dialer

import (
	"context"
	"github.com/Dreamacro/clash/log"
	"github.com/jackpal/gateway"
	"net"

	"github.com/Dreamacro/clash/common/atomic"
	"github.com/Dreamacro/clash/component/resolver"
)

var (
	DefaultOptions     []Option
	DefaultInterface   = atomic.NewTypedValue[string]("")
	DefaultRoutingMark = atomic.NewInt32(0)
)

type NetDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type option struct {
	interfaceName string
	addrReuse     bool
	routingMark   int
	network       int
	prefer        int
	tfo           bool
	mpTcp         bool
	resolver      resolver.Resolver
	netDialer     NetDialer
}

type Option func(opt *option)

func WithInterface(name string) Option {
	return func(opt *option) {
		opt.interfaceName = name
	}
}

func WithAddrReuse(reuse bool) Option {
	return func(opt *option) {
		opt.addrReuse = reuse
	}
}

func WithRoutingMark(mark int) Option {
	return func(opt *option) {
		opt.routingMark = mark
	}
}

func WithResolver(r resolver.Resolver) Option {
	return func(opt *option) {
		opt.resolver = r
	}
}

func WithPreferIPv4() Option {
	return func(opt *option) {
		opt.prefer = 4
	}
}

func WithPreferIPv6() Option {
	return func(opt *option) {
		opt.prefer = 6
	}
}

func WithOnlySingleStack(isIPv4 bool) Option {
	return func(opt *option) {
		if isIPv4 {
			opt.network = 4
		} else {
			opt.network = 6
		}
	}
}

func WithTFO(tfo bool) Option {
	return func(opt *option) {
		opt.tfo = tfo
	}
}

func WithMPTCP(mpTcp bool) Option {
	return func(opt *option) {
		opt.mpTcp = mpTcp
	}
}

func WithNetDialer(netDialer NetDialer) Option {
	return func(opt *option) {
		opt.netDialer = netDialer
	}
}

func WithOption(o option) Option {
	return func(opt *option) {
		*opt = o
	}
}

func GetDefaultInterfaceName() string {
	gateway, err := gateway.DiscoverInterface()
	defaultInterfaceName := DefaultInterface.Load()
	infs, err := net.Interfaces()
	if defaultInterfaceName != "" {
		goto out
	}
	if err != nil {
		log.Errorln("error when discover default interface %+v", err)
		goto out
	}
	for _, inf := range infs {
		addrs, err := inf.Addrs()
		if err != nil {
			log.Errorln("error when step over interface %s %+v", inf.Name, err)
			goto out
		}
		for _, addr := range addrs {
			if ad := addr.(*net.IPNet); ad != nil {
				if ad.IP.String() == gateway.String() {
					defaultInterfaceName = inf.Name
					goto out
				}
			}
		}
	}
out:
	log.Debugln("default interface %s", defaultInterfaceName)
	return defaultInterfaceName
}
