
## build
```
go build -mod vendor

```
## vendor

```bash
go install github.com/goware/modvendor@latest
# vendor不会保留c, dll
go mod vendor
modvendor -copy="**/*.c **/*.h **/*.proto **/*.dll" -v
```

```
sudo dlv debug --listen=:2345 --headless=true -- -d testconfig
```

## Configration example

Configuration example is located at [/docs/config.yaml](https://github.com/MetaCubeX/Clash.Meta/blob/Alpha/docs/config.yaml).

## Docs

Documentation can be found in [Clash.Meta Docs](https://clash-meta.wiki).

## For development

Requirements:
[Go 1.20 or newer](https://go.dev/dl/)

Build Clash.Meta:

```shell
git clone https://github.com/MetaCubeX/Clash.Meta.git
cd Clash.Meta && go mod download
go build
```

Set go proxy if a connection to GitHub is not possible:

```shell
go env -w GOPROXY=https://goproxy.io,direct
```

Build with gvisor tun stack:

```shell
go build -tags with_gvisor
```


### IPTABLES configuration

Work on Linux OS which supported `iptables`

```yaml
# Enable the TPROXY listener
tproxy-port: 9898

iptables:
  enable: true # default is false
  inbound-interface: eth0 # detect the inbound interface, default is 'lo'
```

## Credits

- [Dreamacro/clash](https://github.com/Dreamacro/clash)
- [SagerNet/sing-box](https://github.com/SagerNet/sing-box)
- [riobard/go-shadowsocks2](https://github.com/riobard/go-shadowsocks2)
- [v2ray/v2ray-core](https://github.com/v2ray/v2ray-core)
- [WireGuard/wireguard-go](https://github.com/WireGuard/wireguard-go)
- [yaling888/clash-plus-pro](https://github.com/yaling888/clash)

## License

This software is released under the GPL-3.0 license.

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FDreamacro%2Fclash.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2FDreamacro%2Fclash?ref=badge_large)
