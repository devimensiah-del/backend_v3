package dns

import (
	"context"
	"net"
	"strings"
	"time"
)

type Info struct {
	HasMX       bool     // Do they have email?
	NameServers []string // Who hosts their DNS? (e.g., 'cloudflare', 'aws')
}

func Analyze(ctx context.Context, domain string) Info {
	// Strip protocol if present
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.Split(domain, "/")[0]

	var info Info

	// 1. Check MX Records (Email) - Critical for business validity
	// We use a custom resolver to enforce timeout
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 2 * time.Second,
			}
			return d.DialContext(ctx, network, address)
		},
	}

	mx, err := r.LookupMX(ctx, domain)
	if err == nil && len(mx) > 0 {
		info.HasMX = true
	}

	// 2. Check NameServers (Infrastructure signal)
	ns, err := r.LookupNS(ctx, domain)
	if err == nil {
		for _, n := range ns {
			info.NameServers = append(info.NameServers, n.Host)
		}
	}

	return info
}
