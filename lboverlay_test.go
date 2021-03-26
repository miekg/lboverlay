package lboverlay

import (
	"context"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

// upstreamPlugin is the upstream for overlay.
type upstreamPlugin struct{}

func (u upstreamPlugin) Name() string { return "up" }

func (u upstreamPlugin) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	m := new(dns.Msg)
	m.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeSRV:
		srv1, _ := dns.NewRR("service1.example.com. IN	SRV	0 0 8080 host1.example.com.")
		srv2, _ := dns.NewRR("service1.example.com. IN	SRV	0 0 8080 host2.example.com.")
		m.Answer = []dns.RR{srv1, srv2}

		if state.Do() {
			sig, _ := dns.NewRR("service1.example.com. IN	RRSIG	SRV 8 3 3600 20210423013010 20210319125329 33694 example.com. fvr7Dap1RNTXQ==")
			m.Answer = append(m.Answer, sig)
		}
	case dns.TypeA:
		a1, _ := dns.NewRR("host1.example.com. IN A 127.0.0.1")
		a2, _ := dns.NewRR("host2.example.com. IN A 127.0.0.2")
		m.Answer = []dns.RR{a1, a2}
		if state.Do() {
			sig1, _ := dns.NewRR("host1.example.com. IN	RRSIG	A 8 3 3600 20210423013010 20210319125329 33694 example.com. fvr7Dap1RNTXQ==")
			m.Answer = append(m.Answer, sig1)
			sig2, _ := dns.NewRR("host2.example.com. IN	RRSIG	A 8 3 3600 20210423013010 20210319125329 33694 example.com. fvr7Dap1RNTXQ==")
			m.Answer = append(m.Answer, sig2)
		}
	case dns.TypeAAAA:
		// nodata
		soa, _ := dns.NewRR("example.com. 500 IN SOA ns1.outside.com. root.example.com. 3 604800 86400 2419200 604800")
		m.Extra = []dns.RR{soa}
		if state.Do() {
			sig, _ := dns.NewRR("example.com. IN	RRSIG	SOA 8 2 3600 20210423013010 20210319125329 33694 example.com. fvr7Dap1RNTXQ==")
			m.Answer = append(m.Answer, sig)
		}

	case dns.TypeSOA:
		soa, _ := dns.NewRR("example.com. 500 IN SOA ns1.outside.com. root.example.com. 3 604800 86400 2419200 604800")
		m.Answer = []dns.RR{soa}
		if state.Do() {
			sig, _ := dns.NewRR("example.com. IN	RRSIG	SOA 8 2 3600 20210423013010 20210319125329 33694 example.com. fvr7Dap1RNTXQ==")
			m.Answer = append(m.Answer, sig)
		}
	}

	w.WriteMsg(m)
	return 0, nil
}

var testCases = []test.Case{
	{
		Qname: "service1.example.com.", Qtype: dns.TypeA,
		Answer: []dns.RR{
			test.A("host1.example.com.	303	IN	A	127.0.0.1"),
			test.A("host2.example.com.	303	IN	A	127.0.0.2"),
		},
	},
}

func TestLbOverlay(t *testing.T) {
	o := New("example.com.")
	o.Next = upstreamPlugin{}

	for i, tc := range testCases {
		m := tc.Msg()
		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		_, err := o.ServeDNS(context.TODO(), rec, m)
		if err != nil {
			t.Errorf("Test %d, expected no error, got %s", i, err)
			continue
		}

		resp := rec.Msg
		if err := test.SortAndCheck(resp, tc); err != nil {
			t.Errorf("Test %d: %s", i, err)
		}
	}
}
