package no6

import (
	"context"
	"testing"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

func TestNo6Parse(t *testing.T) {
	type test struct {
		input           string
		shouldErr       bool
		expectedDomains []string
	}
	tests := []*test{
		{
			`no6 {
				.example.com
				example.com
			}
`,
			false, []string{".example.com", "example.com"},
		},
		{
			`no6 {
				invalid.
			}
`,
			true, nil,
		},
		{
			`no6 example.com .example.com`,
			false, []string{".example.com", "example.com"},
		},
		{
			`no6 {
				apiproxy-website-nlb-prod-1-5a4080be4d9bee00.elb.us-east-1.amazonaws.com
			}
`,
			false, []string{"apiproxy-website-nlb-prod-1-5a4080be4d9bee00.elb.us-east-1.amazonaws.com"},
		},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		no6, err := no6Parse(c)

		if err == nil && test.shouldErr {
			t.Fatalf("Test %d expected errors, but got no error", i)
		} else if err != nil && !test.shouldErr {
			t.Fatalf("Test %d expected no errors, but got '%v'", i, err)
		} else if err == nil && !test.shouldErr {
			if no6 == nil {
				t.Fatalf("Test %d: no6 == nil", i)
			}
			if no6.domains == nil {
				t.Fatalf("Test %d: no6.domains == nil", i)
			}
			if test.expectedDomains == nil {
				t.Fatalf("Test %d: text.expectedDomains == nil", i)
			}
			if no6.domains.Len() != len(test.expectedDomains) {
				t.Fatalf("Test %d expected %v, got %v", i, test.expectedDomains, no6.domains.AsSortedSlice())
			}
			for j, name := range no6.domains.AsSortedSlice() {
				if test.expectedDomains[j] != name {
					t.Fatalf("Test %d expected %v for %d th zone, got %v", i, test.expectedDomains[j], j, name)
				}
			}
		}
	}
}

func TestNo6Response(t *testing.T) {
	no6 := New()
	no6.domains.Add("six.example.com", ".ds.example.com")
	no6.Next = &mockHandler{}

	testCases := []test.Case{
		// Answer = what we expect from no6
		// Extra = what the mock returns
		{
			Qname: "four.example.com.",
			Qtype: dns.TypeA,
			Rcode: dns.RcodeSuccess,
			Answer: []dns.RR{
				test.A("four.example.com. 60 IN A 127.0.0.1"),
			},
			Extra: []dns.RR{
				test.A("four.example.com. 60 IN A 127.0.0.1"),
			},
		},
		{
			Qname:  "six.example.com.",
			Qtype:  dns.TypeAAAA,
			Rcode:  dns.RcodeSuccess,
			Answer: []dns.RR{},
			Extra: []dns.RR{
				test.AAAA("six.example.com. 60 IN AAAA ::1"),
			},
		},
		{
			Qname: "six.example.com.",
			Qtype: dns.TypeA,
			Rcode: dns.RcodeSuccess,
			Answer: []dns.RR{
				test.A("six.example.com. 60 IN A 127.0.0.1"),
			},
			Extra: []dns.RR{
				test.A("six.example.com. 60 IN A 127.0.0.1"),
			},
		},
		{
			Qname: "six.example.com.",
			Qtype: dns.TypeANY,
			Rcode: dns.RcodeSuccess,
			Answer: []dns.RR{
				test.A("six.example.com. 60 IN A 127.0.0.1"),
			},
			Extra: []dns.RR{
				test.A("six.example.com. 60 IN A 127.0.0.1"),
				test.AAAA("six.example.com. 60 IN AAAA ::1"),
			},
		},
		{
			Qname:  "sub.ds.example.com.",
			Qtype:  dns.TypeAAAA,
			Rcode:  dns.RcodeSuccess,
			Answer: []dns.RR{},
			Extra: []dns.RR{
				test.AAAA("sub.ds.example.com. 60 IN AAAA ::1"),
			},
		},
		{
			Qname: "sub.ds.example.com.",
			Qtype: dns.TypeANY,
			Rcode: dns.RcodeSuccess,
			Answer: []dns.RR{
				test.A("sub.ds.example.com. 60 IN A 127.0.0.1"),
				test.A("sub.ds.example.com. 60 IN A 127.0.0.2"),
			},
			Extra: []dns.RR{
				test.AAAA("sub.ds.example.com. 60 IN AAAA fe80::1"),
				test.A("sub.ds.example.com. 60 IN A 127.0.0.1"),
				test.AAAA("sub.ds.example.com. 60 IN AAAA fe80::2"),
				test.AAAA("sub.ds.example.com. 60 IN AAAA ::3"),
				test.A("sub.ds.example.com. 60 IN A 127.0.0.2"),
			},
		},
	}

	for i, tc := range testCases {
		no6.Next.(*mockHandler).testCase = tc
		m := tc.Msg()

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		_, err := no6.ServeDNS(context.Background(), rec, m)
		tc.Extra = nil
		if err != nil {
			t.Fatalf("Test %d, expected no error, got %v", i, err)
			return
		}
		if rec.Msg == nil {
			t.Fatalf("Test %d, no message received", i)
		}

		if resp := rec.Msg; rec.Msg != nil {
			if rec.Msg.Rcode != tc.Rcode {
				t.Errorf("Test %d, expected rcode is %d, but got %d", i, tc.Rcode, rec.Msg.Rcode)
				return
			}

			if err := test.SortAndCheck(resp, tc); err != nil {
				t.Errorf("Test %d: %v", i, err)
			}
		}
	}
}

type mockHandler struct {
	testCase test.Case
}

func (h *mockHandler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	r.Answer = nil
	for _, a := range h.testCase.Extra {
		r.Answer = append(r.Answer, a)
	}
	r.Extra = nil

	w.WriteMsg(r)
	return dns.RcodeSuccess, nil
}

func (h *mockHandler) Name() string {
	return "mock"
}
