# lboverlay

## Name

*lboverlay* - use health check data to hand out records.

## Description

The *lboverlay* - load balance overlay - plugin uses health check data (hostname:port:status) that
is overlaid on top of another data source (e.g. from the *file* plugin) to only hand out healthy
addresses to clients. Health check data consists out of "hostname:port" tuples with a health status:
UNKNOWN, UNHEALTHY or HEALTHY.

Health check data is send to coredns process using the DNS protocol. See below for the format of these
packets.

To allow *lboverlay* to match port numbers the data source should contain SRV records that
have those port numbers, e.g. here 8080, and 8081:

    service1.example.org. IN	SRV	0 0 8080 host1.example.org.
    service1.example.org. IN	SRV	0 0 8080 host2.example.org.
    service2.example.org. IN	SRV	0 0 8082 host1.example.org.

And of course the IP addresses of these should also be available in the same zone file/backend.

    host1.example.org. IN	A 127.0.0.1
    host2.example.org. IN	A 127.0.0.2

The above information essentially describes 2 cluster with the following hostnames , IP addresses
and port.

* `service1` with `host1.example.org` (127.0.0.1:8080) and `host2.example.org` (127.0.0.2:8080)
* `service2` with `host1.example.org` (127.0.0.1:8082)

Priority and weight are ignored currently for handing out the SRV records.

The matching *lboverlay* will then work as follows:

1. A health check update is received which says "host1.example.org port 8080" is unhealthy.
2. A query for `service1.example.org. IN A` comes in.
3. *lboverlay* queries the backend for `service1.example.org. IN SRV`.
   * It notes the port numbers in the SRV records.
   * It used SRV target domain (and port number) to map the health check data to and removes unhealthy ones.
   * It resolves the remaining names to A records.
4. Reply with the remaining addresses and set the TTLs to a 5s value.

The *lboverlay* will handle all record types, but the backend MUST return SRV records for things to
work. In case the backend *does not have* SRV records, the original qtype is used to get the data,
it's then let through as-is.

## Syntax

~~~ txt
lboverlay [NAME]
~~~

* where **NAME** is used to as the domain name under which the health checks are reported. Defaults
  to the root domain ".".

## Metrics

If monitoring is enabled (via the *prometheus* plugin) then the following metrics are exported:

* `coredns_lboverlay_healthcheck_total{server}` - Total number of health checks successfully applied.

## Examples

In the following example we have a zone file that contains SRV and A records for the `example.com`
zone saved in a `db.example.com` file.

~~~ dns
service1.example.com. IN	SRV	0 0 8080 host1.example.com.
service1.example.com. IN	SRV	0 0 8080 host2.example.com.
service2.example.com. IN	SRV	0 0 8082 host3.example.com.

host1.example.com. IN A 127.0.0.1
host2.example.com. IN A 127.0.0.2
host3.example.com. IN A 127.0.0.3
~~~

The *lboverlay* plugin can then be configured as follows:

~~~ corefile
example.com {
    file db.example.com
    lboverlay example.com
}
~~~

Then sending a query to CoreDNS, like `dig A service1.example.com` should return:

~~~ dns
;; ANSWER SECTION:
service1.example.com.	5	IN	A	127.0.0.1
service1.example.com.	5	IN	A	127.0.0.2
~~~

Setting the health of host2.example.org port 8080 to unhealthy would remove it from the answer
section.

## Sending Health Checks to *lboverlay*

The health check service will need a list of hosts, ports and a description of how to health check,
how it gets this or how it's formatted is out of scope here.

Health checks can be send to the *lboverlay* plugin, by abusing a DNS request and encoding the
health results in the additional section as SRV records. The TTL is significant to encoding the
health status:

* TTL=0; UKNOWN
* TTL=1; UNHEALTHY
* TTL=2; HEALTHY

Any other TTL values mean the record will be ignored.

The name of the SRV record is set to ".", but this is only to
detect such a request. Potentially this could be signed (TSIG, or RRSIG record) to prevent spoofing
of these updates.

The question section must adhere to: ". IN HINFO" (that's the default, see **NAME**), so the packet
that told *lboverlay* that "host1.example.org port 8080" is unhealthy looks like:

~~~ dns
;; QUESTION SECTION:
;. IN HINFO

;; ADDITIONAL SECTION:
.           1    IN      SRV 0 0 8080 host1.example.org.
~~~

This also means the health checker needs a list of upstream CoreDNS IP addresses and needs to send
the update to all of them.

## See Also

This plugin uses the DNS to receive health checks, another way (and more where the industry is
heading to) is using the xDS protocol from Envoy.

See the *acl* plugin to block health check DNS packets from unwanted sources.

## Bugs

DNSSEC is not supported, as we rewrite ownernames the signatures won't match. I.e. if you backend
is signed, it will break validating clients.

The health check DNS packets are not TSIG signed, which could be an easy way of making sure only
validated health checkers can send updates.
