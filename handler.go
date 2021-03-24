package lboverlay

import (
	"context"

	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Plugin interface.
func (o *Overlay) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// check domains we're repsonbile for
	// check if HC query and update cache (should we discard, or just go with last know good)
	// each should be updated every 10s, so older entries >1h could be removed.
	// check if normal query and do what is described in the README

	return 0, nil
}
