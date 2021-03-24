# lboverlay

## Name

*lboverlay* - use health check data to hand out records.

## Description

TODO; is the focus IP:port or hostname:port. If hostname, we don't have have to encode the IP
addresses again in the health checker (it can get them from the DNS)

The *lboverlay* - load balance overlay - plugin uses health check data (hostname:port:status) that
is overlaid on top of another data source (e.g. from the *file* plugin) to only hand out healthy
addresses to clients. Health check data consists out of "hostname:port" tuples with a health status:
UNKNOWN, UNHEALTHY or HEALTHY.

Health check data is send to coredns using the DNS protocol. See below for the format of these
packets.

To allow *lboverlay* to match port numbers the data source should contain SRV records that
have those port numbers, e.g. here 8080, and 8081:

    service1.example.org. IN	SRV	0 0 8080 host1.example.org.
    service1.example.org. IN	SRV	0 0 8080 host2.example.org.
    service2.example.org. IN	SRV	0 0 8082 host3.example.org.

And of course the IP addresses of these should also be available in the same zone file/backend.

    host1.example.org. IN	A 127.0.0.1
    host2.example.org. IN	A 127.0.0.2
    host3.example.org. IN	A 127.0.0.3

The above information essentially describes 2 cluster with the following IP:ports.

* `service1` with 127.0.0.1:8080 and 127.0.0.2:8080
* `service2` with 127.0.0.3:8082

Priority and weight are ignored currently for handing out the SRV records.

The matching *lboverlay* will do will then work as follows:

1. A health check update is received which says "host1.example.org port 8080" is unhealthy.
2. A query for `service1.example.org. IN A` comes in.
3. *lboverlay* queries the backend for `service1.example.org. IN SRV`.
   * It notes the port numbers in the SRV records.
   * It used SRV target domain to map the health check data to and removes unhealthy ones.
   * It resolves the remaining names to A records.
4. Reply with the remaining addresses..

The *lboverlay* will handle the following record types and will use the health check data: A, AAAA,
MX, and SRV.

In case the backend _does not have_ SRV records, the original qtype is used to get the data, it's
then let through as-is.

## Syntax

~~~ corefile
lboverlay [ADDRESS]
~~~

* **ADDRESS** is the address to listen on. Defaults to TBD.

## Sending Health Checks to *lboverlay*

The health check service will need a list of hosts, ports and a description of how to health check,
how it gets this or how it's formatted is out of scope here.

Health checks can be send to the *lboverlay* plugin, by abusing the a DNS request and encoding the
health results in the additional section as SRV records. The TTL is then significant to encoding the
health status:

* TTL=0; UKNOWN
* TTL=1; UNHEALTHY
* TTL=2; HEALTHY

The name of the SRV record is set to "." (the root domain) as it's not significant.

The question section must adhere to: "lboverlay.coredns.io IN SRV", so the packet that told
*lboverlay* that "host1.example.org port 8080" is unhealthy looks like:

~~~ dns
;; QUESTION SECTION:
;lboverlay.coredns.io IN SRV

;; ADDITIONAL SECTION:
.           1    IN      SRV 0 0 8080 host1.example.org.
~~~

This also means the health checker needs a list of upstream CoreDNS IP addresses and need to send
the update to all of them.

## See Also
