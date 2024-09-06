package no6

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/nonwriter"
	"github.com/miekg/dns"
	"go.fuhry.dev/runtime/utils/hashset"
)

var log = clog.NewWithPlugin("no6")

var regexpValidDomain = regexp.MustCompile(`^\.?[a-z0-9-]+(\.[a-z0-9]+)*$`)

func init() {
	plugin.Register("no6", setup)
}

func setup(c *caddy.Controller) error {
	s, err := no6Parse(c)
	if err != nil {
		return plugin.Error("no6", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		s.Next = next
		return s
	})
	return nil
}

type No6 struct {
	Next plugin.Handler

	origins *hashset.HashSet[string]
	domains *hashset.HashSet[string]
}

func New() *No6 {
	s := &No6{
		domains: hashset.NewHashSet[string](),
		origins: hashset.NewHashSet[string](),
	}

	return s
}

func no6Parse(c *caddy.Controller) (*No6, error) {
	s := New()

	i := 0
	for c.Next() {
		if i > 0 {
			return s, plugin.ErrOnce
		}
		i++

		args := c.RemainingArgs()
		if len(args) > 0 {
			// single-line format
			for _, arg := range args {
				if err := s.addDomain(arg); err != nil {
					return nil, err
				}
			}
		} else {
			// block format
			for c.NextBlock() {
				if err := s.addDomain(c.Val()); err != nil {
					return nil, err
				}
				if len(c.RemainingArgs()) > 0 {
					return nil, fmt.Errorf("parsing block failed: line %d: no extra arguments allowed", i)
				}
			}
		}
	}

	return s, nil
}

func (s *No6) addDomain(v string) error {
	if !regexpValidDomain.MatchString(v) {
		return fmt.Errorf("parsing block failed: %q: not a valid domain or domain suffix", v)
	}

	s.domains.Add(v)
	return nil
}

func (s *No6) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	nw := nonwriter.New(w)
	rcode, err := plugin.NextOrFailure(s.Name(), s.Next, ctx, nw, r)
	if err != nil {
		if nw.Msg != nil {
			w.WriteMsg(nw.Msg)
		}
		return rcode, err
	}

	r = nw.Msg
	if r == nil {
		return dns.RcodeServerFailure, fmt.Errorf("no answer received")
	}

	var v4, v6, questionAAAA bool

	for _, q := range r.Question {
		if q.Qtype == dns.TypeAAAA {
			questionAAAA = true
			break
		}
	}

	if rcode == dns.RcodeSuccess {
		var remove sort.IntSlice
		for i, ans := range r.Answer {
			if aaaa, ok := ans.(*dns.AAAA); ok {
				v6 = true
				filter := false
				trimmedName := strings.TrimSuffix(aaaa.Header().Name, ".")
				if s.domains.Contains(trimmedName) {
					filter = true
				}
				if !filter {
					for _, domain := range s.domains.AsSlice() {
						if len(domain) >= 1 && domain[0] == '.' && strings.HasSuffix(trimmedName, domain) {
							filter = true
							break
						}
					}
				}

				if filter {
					remove = append(remove, i)
				}
			} else if _, ok := ans.(*dns.A); ok {
				v4 = true
			}
		}

		// only remove AAAA responses if either the question was for AAAA only, or
		// it's a dual stack response with A and AAAA records.
		if questionAAAA || (v4 && v6) {
			// if we are supposed to remove everything from the answer, just set it to nil
			if len(remove) == len(r.Answer) {
				r.Answer = nil
			} else {
				// we track items to remove in index order, so each item removed decrements the
				// index of each later item by 1 - compensate for this by subtracting i from j to
				// calculate the modified index after previous items were removed
				for i, j := range remove {
					idx := j - i
					r.Answer = slices.Delete(r.Answer, idx, idx+1)
				}
			}
		}
	}

	w.WriteMsg(r)

	return rcode, err
}

func (s *No6) Name() string {
	return "no6"
}
