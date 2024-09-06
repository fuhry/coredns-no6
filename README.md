# no6: Block IPv6 resolution of certain domains

`no6` is a coredns plugin that selectively blocks IPv6 name resolution for a user-configured list of domains.

## Why?

My ISP does not offer native IPv6 access, so for many years (since high school, 2008-2009!) I have used Hurricane
Electric's tunnel service, [tunnelbroker.net](https://tunnelbroker.net/). Starting in mid 2024 I noticed I was
getting harder reCAPTCHA challenges, being blocked from watching youtube videos, etc., apparently because Google has
now classified all IPv6 tunnel brokers as bot-abuse subnets.

Training Google's AI to identify buses and crosswalks was mildly annoying for me, and I was able to work around it with
some effort: I primarily browse with Firefox these days, and Firefox has a config setting `network.dns.ipv4OnlyDomains`
that can be used to force IPv4 for reCAPTCHA-related domains. But my wife uses Chrome, which does not have an equivalent
setting. Moreover, the Firefox config setting does not scale: it has to be manually configured on every host or deployed 
through config management, so that's not great either, because the reCAPTCHA issue was affecting everything on the
network, including smart TVs, mobile devices, etc.

Some large tech companies offer alternate DNS servers that do things like block mature content, don't send AAAA 
responses, etc., but I wanted a generic way to accomplish this without having to look up alternate nameservers for a
slew of domains.

And thus, `no6` was born.

## How to use

Enabling `no6` in coredns is quite straightforward:

* Add `no6:github.com/fuhry/coredns-no6` to `plugin.cfg`
* Run `make`

## How to configure

Configuration is similarly simple: just list the domains which should block IPv6 resolution as arguments to the plugin
or inside the configuration block.

```
no6 example.com
# or
no6 {
    example.com
}
```

Use the syntax `.example.com` (with a leading period) to block AAAA answers for all names _ending in_ `.example.com`.
To block the apex domain, you'll need to use a separate entry.

## Technical details

`no6` matches on names in the answer, not in the question. So, if your blocklist contains `example.com` but a query for
`example.com` returns an answer for `othersite.com` (due to a CNAME, etc.), that answer won't be filtered.

An AAAA record is filtered out of the answer if:

1. The rcode is `NOERROR`; and
2. The record's name matches the domain allowlist; and either:
  * The question type is `AAAA`; or
  * The question type is `ANY` _and_ the response contains at least one `A` record answer

In practice, most clients don't use `ANY` queries, nor do many servers support them, so `no6` will spend most of its
time just stripping out addresses from `AAAA` responses.

## Quick start: how to end the reCAPTCHA/youtube pain

```
no6 {
        .google.com
        .gstatic.com
        .googleapis.com
        .googletagmanager.com
        .googlevideo.com
        .youtube.com
}
```

# Author/Contact

Dan Fuhry <dan@fuhry.com>

&copy; 2024. Released under the Apache 2.0 License; see the [LICENSE](LICENSE) file for details.