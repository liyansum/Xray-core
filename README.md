# Xray-core for XrayR

This repository is a reduced Xray-core build intended for the paired XrayR
repository. It keeps only the runtime features used by that deployment.

## Supported scope

- Inbound protocols: Trojan, Shadowsocks/SS2022, SOCKS and HTTP/mixed.
- Outbound protocols: Shadowsocks/SS2022, SOCKS, direct and block.
- Transports: raw TCP, UDP and TLS.
- Services: routing, DNS including DoH/DoQ, FakeDNS, policy, statistics and
  observatory support required by XrayR.
- Traffic sniffing and the registered protocol sniffers remain available.

VMess, VLESS, REALITY, WebSocket, gRPC transport, mKCP/KCP, HTTPUpgrade,
XHTTP/SplitHTTP, TUN, WireGuard, Hysteria, FinalMask, reverse proxy, the gRPC
command API and SS-Plugin support are intentionally removed.

## Build

The paired XrayR release build uses `GOAMD64=v3` and `-ldflags="-s -w"`.
GOAMD64 v3 binaries require an x86-64-v3 capable CPU.

This project is derived from [XTLS/Xray-core](https://github.com/XTLS/Xray-core)
and remains licensed under the Mozilla Public License 2.0. See `LICENSE`.
