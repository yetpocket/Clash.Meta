package dialer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Dreamacro/clash/component/resolver"
	"github.com/Dreamacro/clash/log"
)

type dialFunc func(ctx context.Context, network string, ips []netip.Addr, port string, opt *option) (net.Conn, error)

var (
	dialMux                      sync.Mutex
	actualSingleStackDialContext = serialSingleStackDialContext
	actualDualStackDialContext   = serialDualStackDialContext
	tcpConcurrent                = false
	fallbackTimeout              = 300 * time.Millisecond
)

func applyOptions(options ...Option) *option {
	opt := &option{
		interfaceName: DefaultInterface.Load(),
		routingMark:   int(DefaultRoutingMark.Load()),
	}

	for _, o := range DefaultOptions {
		o(opt)
	}

	for _, o := range options {
		o(opt)
	}

	return opt
}

func DialContext(ctx context.Context, network, address string, options ...Option) (net.Conn, error) {
	opt := applyOptions(options...)

	if opt.network == 4 || opt.network == 6 {
		if strings.Contains(network, "tcp") {
			network = "tcp"
		} else {
			network = "udp"
		}

		network = fmt.Sprintf("%s%d", network, opt.network)
	}

	ips, port, err := parseAddr(ctx, network, address, opt.resolver)
	if err != nil {
		return nil, err
	}

	switch network {
	case "tcp4", "tcp6", "udp4", "udp6":
		return actualSingleStackDialContext(ctx, network, ips, port, opt)
	case "tcp", "udp":
		return actualDualStackDialContext(ctx, network, ips, port, opt)
	default:
		return nil, ErrorInvalidedNetworkStack
	}
}

func ListenPacket(ctx context.Context, network, address string, options ...Option) (net.PacketConn, error) {
	cfg := applyOptions(options...)

	lc := &net.ListenConfig{}
	if cfg.interfaceName != "" {
		addr, err := bindIfaceToListenConfig(cfg.interfaceName, lc, network, address)
		if err != nil {
			return nil, err
		}
		address = addr
	}
	if cfg.addrReuse {
		addrReuseToListenConfig(lc)
	}
	if cfg.routingMark != 0 {
		bindMarkToListenConfig(cfg.routingMark, lc, network, address)
	}

	return lc.ListenPacket(ctx, network, address)
}

func SetTcpConcurrent(concurrent bool) {
	dialMux.Lock()
	defer dialMux.Unlock()
	tcpConcurrent = concurrent
	if concurrent {
		actualSingleStackDialContext = concurrentSingleStackDialContext
		actualDualStackDialContext = concurrentDualStackDialContext
	} else {
		actualSingleStackDialContext = serialSingleStackDialContext
		actualDualStackDialContext = serialDualStackDialContext
	}
}

func GetTcpConcurrent() bool {
	dialMux.Lock()
	defer dialMux.Unlock()
	return tcpConcurrent
}

func dialContext(ctx context.Context, network string, destination netip.Addr, port string, opt *option) (net.Conn, error) {
	address := net.JoinHostPort(destination.String(), port)

	netDialer := opt.netDialer
	switch netDialer.(type) {
	case nil:
		netDialer = &net.Dialer{}
	case *net.Dialer:
		_netDialer := *netDialer.(*net.Dialer)
		netDialer = &_netDialer // make a copy
	default:
		return netDialer.DialContext(ctx, network, address)
	}

	dialer := netDialer.(*net.Dialer)
	if opt.interfaceName != "" {
		if err := bindIfaceToDialer(opt.interfaceName, dialer, network, destination); err != nil {
			return nil, err
		}
	}
	if opt.routingMark != 0 {
		bindMarkToDialer(opt.routingMark, dialer, network, destination)
	}
	if opt.mpTcp {
		setMultiPathTCP(dialer)
	}
	if opt.tfo {
		return dialTFO(ctx, *dialer, network, address)
	}
	return dialer.DialContext(ctx, network, address)
}

func serialSingleStackDialContext(ctx context.Context, network string, ips []netip.Addr, port string, opt *option) (net.Conn, error) {
	return serialDialContext(ctx, network, ips, port, opt)
}

func serialDualStackDialContext(ctx context.Context, network string, ips []netip.Addr, port string, opt *option) (net.Conn, error) {
	return dualStackDialContext(ctx, serialDialContext, network, ips, port, opt)
}

func concurrentSingleStackDialContext(ctx context.Context, network string, ips []netip.Addr, port string, opt *option) (net.Conn, error) {
	return parallelDialContext(ctx, network, ips, port, opt)
}

func concurrentDualStackDialContext(ctx context.Context, network string, ips []netip.Addr, port string, opt *option) (net.Conn, error) {
	if opt.prefer != 4 && opt.prefer != 6 {
		return parallelDialContext(ctx, network, ips, port, opt)
	}
	return dualStackDialContext(ctx, parallelDialContext, network, ips, port, opt)
}

func dualStackDialContext(ctx context.Context, dialFn dialFunc, network string, ips []netip.Addr, port string, opt *option) (net.Conn, error) {
	ipv4s, ipv6s := resolver.SortationAddr(ips)
	if len(ipv4s) == 0 && len(ipv6s) == 0 {
		return nil, ErrorNoIpAddress
	}

	preferIPVersion := opt.prefer
	fallbackTicker := time.NewTicker(fallbackTimeout)
	defer fallbackTicker.Stop()

	results := make(chan dialResult)
	returned := make(chan struct{})
	defer close(returned)

	var wg sync.WaitGroup

	racer := func(ips []netip.Addr, isPrimary bool) {
		defer wg.Done()
		result := dialResult{isPrimary: isPrimary}
		defer func() {
			select {
			case results <- result:
			case <-returned:
				if result.Conn != nil && result.error == nil {
					_ = result.Conn.Close()
				}
			}
		}()
		result.Conn, result.error = dialFn(ctx, network, ips, port, opt)
	}

	if len(ipv4s) != 0 {
		wg.Add(1)
		go racer(ipv4s, preferIPVersion != 6)
	}

	if len(ipv6s) != 0 {
		wg.Add(1)
		go racer(ipv6s, preferIPVersion != 4)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var fallback dialResult
	var errs []error

loop:
	for {
		select {
		case <-fallbackTicker.C:
			if fallback.error == nil && fallback.Conn != nil {
				return fallback.Conn, nil
			}
		case res, ok := <-results:
			if !ok {
				break loop
			}
			if res.error == nil {
				if res.isPrimary {
					return res.Conn, nil
				}
				fallback = res
			} else {
				if res.isPrimary {
					errs = append([]error{fmt.Errorf("connect failed: %w", res.error)}, errs...)
				} else {
					errs = append(errs, fmt.Errorf("connect failed: %w", res.error))
				}
			}
		}
	}

	if fallback.error == nil && fallback.Conn != nil {
		return fallback.Conn, nil
	}
	return nil, errors.Join(errs...)
}

func parallelDialContext(ctx context.Context, network string, ips []netip.Addr, port string, opt *option) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, ErrorNoIpAddress
	}
	results := make(chan dialResult)
	returned := make(chan struct{})
	defer close(returned)
	racer := func(ctx context.Context, ip netip.Addr) {
		result := dialResult{isPrimary: true, ip: ip}
		defer func() {
			select {
			case results <- result:
			case <-returned:
				if result.Conn != nil && result.error == nil {
					_ = result.Conn.Close()
				}
			}
		}()
		result.Conn, result.error = dialContext(ctx, network, ip, port, opt)
	}

	for _, ip := range ips {
		go racer(ctx, ip)
	}
	var errs []error
	for i := 0; i < len(ips); i++ {
		res := <-results
		if res.error == nil {
			return res.Conn, nil
		}
		errs = append(errs, res.error)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return nil, os.ErrDeadlineExceeded
}

func serialDialContext(ctx context.Context, network string, ips []netip.Addr, port string, opt *option) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, ErrorNoIpAddress
	}
	var errs []error
	for _, ip := range ips {
		if conn, err := dialContext(ctx, network, ip, port, opt); err == nil {
			return conn, nil
		} else {
			errs = append(errs, err)
		}
	}
	return nil, errors.Join(errs...)
}

type dialResult struct {
	ip netip.Addr
	net.Conn
	error
	isPrimary bool
}

func parseAddr(ctx context.Context, network, address string, preferResolver resolver.Resolver) ([]netip.Addr, string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, "-1", err
	}
	server := ""
	// preferResolver != nil 只是检查接口不是nil
	// 但里面的值可能是nil
	// https://stackoverflow.com/questions/13476349/check-for-nil-and-nil-interface-in-go
	// https://go.dev/play/p/Isoo0CcAvr
	if preferResolver != nil {
		server = preferResolver.Addr()
	}
	log.Debugln("trying resolve network %s address %s. dns server %s", network, address, server)
	var ips []netip.Addr
	switch network {
	case "tcp4", "udp4":
		if preferResolver == nil {
			ips, err = resolver.LookupIPv4ProxyServerHost(ctx, host)
		} else {
			ips, err = resolver.LookupIPv4WithResolver(ctx, host, preferResolver)
		}
	case "tcp6", "udp6":
		if preferResolver == nil {
			ips, err = resolver.LookupIPv6ProxyServerHost(ctx, host)
		} else {
			ips, err = resolver.LookupIPv6WithResolver(ctx, host, preferResolver)
		}
	default:
		if preferResolver == nil {
			log.Debugln("lookup proxy server [%s] ip address", address)
			ips, err = resolver.LookupIPProxyServerHost(ctx, host)
		} else {
			log.Debugln("lookup proxy server [%s] by resolver", address)
			ips, err = resolver.LookupIPWithResolver(ctx, host, preferResolver)
		}
	}

	if err != nil {
		return nil, "-1", fmt.Errorf("dns resolve failed: %w", err)
	}
	for i, ip := range ips {
		if ip.Is4In6() {
			ips[i] = ip.Unmap()
		}
	}
	return ips, port, nil
}

type Dialer struct {
	Opt option
}

func (d Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return DialContext(ctx, network, address, WithOption(d.Opt))
}

func (d Dialer) ListenPacket(ctx context.Context, network, address string, rAddrPort netip.AddrPort) (net.PacketConn, error) {
	opt := WithOption(d.Opt)
	if rAddrPort.Addr().Unmap().IsLoopback() {
		// avoid "The requested address is not valid in its context."
		opt = WithInterface("")
	}
	return ListenPacket(ctx, ParseNetwork(network, rAddrPort.Addr()), address, opt)
}

func NewDialer(options ...Option) Dialer {
	opt := applyOptions(options...)
	return Dialer{Opt: *opt}
}
