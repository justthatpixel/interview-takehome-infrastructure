package store

import (
	"net/netip"

	"github.com/trybyteful/interview-takehome/services/api/internal/services/proxy"
)

type proxyRow struct {
	IP       netip.Addr
	Port     int32
	Country  string
	City     string
	Protocol string
}

func (r proxyRow) toEntity() proxy.Proxy {
	return proxy.Proxy{
		Addr:     netip.AddrPortFrom(r.IP, uint16(r.Port)),
		Country:  r.Country,
		City:     r.City,
		Protocol: r.Protocol,
	}
}
