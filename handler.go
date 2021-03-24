package lboverlay

import (
	"context"

	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Plugin interface.
func (o *Overlay) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// check domains we're repsonbile for
	// check if HC query
	// check if normal query

	return 0, nil
}
