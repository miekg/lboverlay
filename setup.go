package lboverlay

import (
	"fmt"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/miekg/dns"
)

func init() { plugin.Register("lboverlay", setup) }

func setup(c *caddy.Controller) error {
	hcname, err := parse(c)
	if err != nil {
		return plugin.Error("lboverlay", err)
	}

	o := New(hcname)
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		o.Next = next
		return o
	})
	return nil
}

func parse(c *caddy.Controller) (string, error) {
	for c.Next() {
		args := c.RemainingArgs()

		switch len(args) {
		case 0:
			return "", nil
		case 1:
			if _, ok := dns.IsDomainName(args[0]); !ok {
				return "", fmt.Errorf("not a domain name: %s", args[0])
			}
			return args[0], nil
		default:
			return "", c.ArgErr()
		}
	}
	return "", nil
}
