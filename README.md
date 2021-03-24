# lboverlay

## Name

*lboverlay* - use health check data to hand out IP addresses.

## Description

The *lboverlay* - load balance overlay - plugin uses health check data that is overlaid on top of
any other data source (e.g. from the *file* plugin) to only hand out healthy addresses to clients.
Health check data consists out of "hostname:port" tuples with a health state: UKNNOWN, UNHEALTHY or
HEALTHY.

Health check data is send to coredns using the DNS protocol. See below for the format of these
packets.

To allow *lboverlay* to match port numbers the data source should contain SRV records that
have those port numbers, here 8080, and 8081:

    service1.example.org. IN	SRV	0 0 8080 host1.example.org.
    service1.example.org. IN	SRV	0 0 8080 host2.example.org.
    service2.example.org. IN	SRV	0 0 8082 host3.example.org.

And of course the IP addresses of these should also be available in the same zonefile.

    host1.example.org. IN	A 127.0.0.1
    host2.example.org. IN	A 127.0.0.2
    host3.example.org. IN	A 127.0.0.3

The matching *lboverlay* will do will then work as follows:

1. A healt hcheck update is received which says "host1.example.org port 8080" is unhealthy.
2. A query for `service1.example.org. IN A` comes in.
3. *lboverlay* queries the backend for `service1.example.org. IN SRV`.
   * It notes the port numbers in the SRV records.
   * It used SRV target domain to map the health check data to.
   * It builds a list of `<hostname:port>`.
   * It filters each `<hostname:port>` tuple.
   * It resolves the hostname to A records
4. Reply with the remaining addresses.

## Syntax

~~~ corefile
lboverlay [ADDRESS]
~~~

* **ADDRESS** is the address to listen on. Defaults to TBD.

## Health Check Description

Because of the close connection between what should be health checked and what data should be handed
out by *lboverlay* it could make some sense to share the zone file (as described above) with the
health check service. Except that a health checker would need additional data that doesn't "fit" in
a zone file. To work around this you can specify comments in the zonfile that tell the checker what
kind of health check to perform.

These comments have the format of: key=value pairs, where the following have been defined:

* `proto=udp|tcp|grpc|http`
* `timeout=DURATION`, default to a 5s timeout if not given.

These are separated by commas (no spaces), e.g:

    service1.example.org. IN	SRV	0 0 8080 host1.example.org. ; proto=tcp,timeout=5s
    service1.example.org. IN	SRV	0 0 8080 host2.example.org. ; proto=udp,timeout=3s
    service2.example.org. IN	SRV	0 0 8082 host3.example.org. ; proto=http


TODO(miek): how to get the IP address? Search?? What about multiple addresses for a host, say only
v4 and v6? Or take the names as normative for HC purpose?

## Sending Health Checks to *lboverlay*

Health checks can be send to the *lboverlay* plugin, by abusing the a DNS request and encoding the
health results in the additional section as SRV records. The TTL is then significant to encoding the
health status:

* TTL=0; UKNOWN
* TTL=1; UNHEALTHY
* TTL=2; HEALTHY

The question section must adhere to: "lboverlay.coredns.io IN SRV".


## See Also

See <https://github.com/miekg/dns> for a zone parser that returns comments from zone files.
