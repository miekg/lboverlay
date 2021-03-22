# lboverlay

## Name

*lboverlay* - use health check data to hand out IP addresses.

## Description

The *lboverlay* - load balance overlay - plugin uses health check data that is overlaid on top of
any other data source (e.g. from the *file* plugin) to only hand out healthy addresses to clients.

Health check data is send to coredns using the xDS protocol. This protocol uses *clusters* that hold
endpoints (IP + port). IPs can be easily matched to A and AAAA records, but port numbers are
problematic. If 2 services share the same IP address, and only 1 is un healthy this would imply that
*lboverlay* can't hand out that A/AAAA record; meaning from that standpoint they will be both
unhealthy.

To allow *lboverlay* to _also_ match port numbers the data source should contain SRV records that
have those port numbers, here 8080, and 8081.

    service1.example.org. IN	SRV	0 0 8080 host1.example.org.
    service1.example.org. IN	SRV	0 0 8080 host2.example.org.
    service2.example.org. IN	SRV	0 0 8082 host3.example.org.

And of course the IP addresses of these should also be available:

    host1.example.org. IN	A 127.0.0.1
    host2.example.org. IN	A 127.0.0.2
    host3.example.org. IN	A 127.0.0.3

The matching *lboverlay* will do will then work as follows:

1. A healt hcheck update is received which says "127.0.0.1 port 8080" is unhealthy.
2. A query for `service1.example.org. IN A` comes in.
3. *lboverlay* queries the backend for `service1.example.org. IN SRV`.
   * It notes the port numbers in the SRV records.
   * It resolves the SRV target domains to IP addresses (A or AAAA).
   * It builds a list of `<ip:port>`.
   * It filters each `<ip:port>` tuple.
4. Reply with the remaining addresses.

## Syntax

~~~ corefile
lboverlay [ADDRESS] {
    tls CERT KEY CA
    tls_servername NAME
}
~~~

* **ADDRESS** is the address to listen on. Defaults to TBD.
* `tls` and `tls_servername` are explained in, e.g. the *forward* and *grpc* plugins.

## See Also
