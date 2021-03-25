package lboverlay

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Plugin interface.
func (o *Overlay) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// each HC entity should be updated every 10s, so older entries >1h could be removed. TODO(miek)
	state := request.Request{W: w, Req: r}

	if o.isHealthCheck(state) {
		for _, rr := range r.Extra {
			srv, ok := rr.(*dns.SRV)
			if !ok {
				log.Debugf("Non SRV record in health check: %s", rr)
				continue
			}
			log.Debugf("Health status for %q set to: %s", joinHostPort(srv.Target, srv.Port), status(srv.Header().Ttl))
			o.setStatus(srv, status(srv.Header().Ttl))
		}
		resp := new(dns.Msg)
		resp.SetReply(r)
		w.WriteMsg(resp)
		return 0, nil
	}

	// if we see a SRV reply, we wrap it on our response writer and filter out the baddies in our own responseWriter
	if state.QType() == dns.TypeSRV {
		ow := &ResponseWriter{w, o}
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, ow, r)
	}

	resp, err := o.u.Lookup(ctx, state, state.Name(), dns.TypeSRV)
	if err != nil {
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, w, r)
	}
	// check the response beforehand to make code below simpler because less corner cases.
	srvs := 0
	for _, rr := range resp.Answer {
		if _, ok := rr.(*dns.SRV); ok {
			srvs++
		}
	}
	if srvs == 0 || len(resp.Answer) != srvs { // the response doesn't have (enough) SRV in it, call NextOrFailure and be a noop
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, w, r)
	}

	// check what we have, as we should have SRVs, but might not have all A/AAAA/MX records
	healthySRVs := make([]*dns.SRV, 0, len(resp.Answer))
	for _, rr := range resp.Answer {
		srv := rr.(*dns.SRV)
		s := o.status(srv)
		log.Debugf("Health status for %q is: %s", joinHostPort(srv.Target, srv.Port), s)
		if s == statusUnhealthy {
			continue
		}
		healthySRVs = append(healthySRVs, srv)
	}

	// for the healthy SRVs we need to resolve the target names with the original qtype from the query
	m := new(dns.Msg)
	m.SetReply(r)
	m.Answer = make([]dns.RR, 0, len(healthySRVs))
	for _, srv := range healthySRVs {
		// inspecting the additional section above might alleviate the extra queries here. TODO(miek)
		resp, err := o.u.Lookup(ctx, state, srv.Target, state.QType())
		if err != nil {
			continue
		}
		log.Debugf("Found answer for %s/%d, adding %d record(s)", srv.Target, state.QType(), len(resp.Answer))
		for _, rr := range resp.Answer {
			rr.Header().Name = state.QName()
			m.Answer = append(m.Answer, rr)
		}
	}
	// nodata, nxdomain and the like. TODO
	// SOA query from backend to at least be able to get that?
	// How about RRSIG and the like, not handled.

	w.WriteMsg(m)

	return dns.RcodeSuccess, nil
}
