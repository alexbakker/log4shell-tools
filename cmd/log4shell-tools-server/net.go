package main

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"inet.af/netaddr"
)

func getRemoteAddr(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		ips := strings.Split(r.Header.Get(header), ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			realIP, err := netaddr.ParseIP(ip)
			if err != nil || realIP.IsPrivate() {
				continue
			}
			return ip
		}
	}
	return r.RemoteAddr
}

func getAddrPtr(ctx context.Context, addr string) (string, *string) {
	resCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	var ptr *string
	var resolver net.Resolver
	if names, err := resolver.LookupAddr(resCtx, host); err == nil && len(names) != 0 {
		ptr = &names[0]
	}

	return host, ptr
}
