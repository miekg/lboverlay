package main

import (
	"context"
	"testing"

	"github.com/miekg/dns"
)

type upstreamPlugin struct{}

func (u upstreamPlugin) Name() string { return "up" }

func (u upstreamPlugin) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeSRV:
		srv1, _ := dns.NewRR("service1.example.com. IN	SRV	0 0 8080 host1.example.com.")
		srv2, _ := dns.NewRR("service1.example.com. IN	SRV	0 0 8080 host2.example.com.")
		m.Answer = []dns.RR{srv1, srv2}
	case dns.TypeA:
		a1, _ := dns.NewRR("host1.example.com. IN A 127.0.0.1")
		a2, _ := dns.NewRR("host2.example.com. IN A 127.0.0.2")
		m.Answer = []dns.RR{a1, a2}
	case dns.TypeAAAA:
		// nodata
		soa, _ := dns.NewRR("example.com. 500 IN SOA ns1.outside.com. root.example.com. 3 604800 86400 2419200 604800")
		m.Extra = []dns.RR{soa}

	case dns.TypeSOA:
		soa, _ := dns.NewRR("example.com. 500 IN SOA ns1.outside.com. root.example.com. 3 604800 86400 2419200 604800")
		m.Answer = []dns.RR{soa}
	}

	w.WriteMsg(m)
}

func TestLbOverlay(t *testing.T) {
	// t.Fatal("not implemented")
}
