# log4shell.tools [![build](https://github.com/alexbakker/log4shell-tools/actions/workflows/build.yml/badge.svg)](https://github.com/alexbakker/log4shell-tools/actions/workflows/build.yml)

__log4shell.tools__ is a tool that allows you to run a test to check whether one
of your applications is affected by the recent vulnerabilities in log4j:
__CVE-2021-44228__ and __CVE-2021-45046__.

This is the code that runs https://log4shell.tools. If you'd like to inspect the
code or run an instance in your own environment, you've come to the right
place.

## How does this work?

The tool generates a unique ID for you to test with. After you click start,
we'll generate a piece of text for you that looks similar to this:
__${jndi:ldap://\*.dns.log4shell.tools:12345/\*}__. Copy it and paste it anywhere
you suspect it might end up getting passed through log4j. For example: search
boxes, form fields or HTTP headers.

Once an outdated version of log4j sees this string, it will perform a DNS lookup
to get the IP address of __\*.dns.log4shell.tools__. If this happens, it is
considered the first sign of vulnerability to information leakage. Next, it will
attempt and LDAP search request to __log4shell.tools:12345__. The tool responds
with a Java class description, along with a URL for where to obtain it. Log4j
may even attempt to fetch the class file. The tool will return a 404 and
conclude the test.

## Screenshot

<img width="750" src="https://alexbakker.me/u/iq8qmxclfb.png"/>

## Installation

The tool was tested with Go 1.16. Make sure it (or a more recent version of Go) is
installed and run the following command:

```
go install github.com/alexbakker/log4shell-tools/cmd/log4shell-tools-server
```

The binary will be available in ``$GOPATH/bin``

### Usage

Since this tool compiles to a single binary, all you have to do is run it to
start self hosting an instance of log4shell.tools. To make it accessible by
other machines in your network, you'll want to pass a couple of flags to stop
the tool from only listening on the loopback interface. If you're exposing this
to the internet, you'll probably also want to put a reverse proxy in front of
the HTTP server. Ignore the DNS options for now, they're not needed for simple
internal deployments.

For the full list of available flags, run `log4shell-tools-server -h`:

```
Usage of ./log4shell-tools-server:

This tool only listens on 127.0.0.1 by default. Pass the flags below to customize for your environment.

  -dns-a string
    	the IPv4 address to respond with to any A record queries for 'dns-zone' (default "127.0.0.1")
  -dns-aaaa string
    	the IPv6 address to respond with to any AAAA record queries for 'dns-zone' (default "::1")
  -dns-addr string
    	listening address for the DNS server (default "127.0.0.1:12346")
  -dns-enable
    	enable the DNS server
  -dns-zone string
    	DNS zone that is forwarded to the tool's DNS server (example: "dns.log4shell.tools")
  -http-addr string
    	listening address for the HTTP server (default "127.0.0.1:8001")
  -http-addr-external string
    	address where the HTTP server can be reached externally (default "127.0.0.1:8001")
  -ldap-addr string
    	listening address for the LDAP server (default "127.0.0.1:12345")
  -ldap-addr-external string
    	address where the LDAP server can be reached externally (default "127.0.0.1:12345")
  -ldap-http-proto string
    	the HTTP protocol to use in the payload URL that the LDAP server responds with (default "https")
  -storage string
    	storage connection URI (either memory:// or a postgres:// URI (default "memory://")
  -test-timeout int
    	test timeout in minutes (default 30)
```

#### Storage

The tool uses its in-memory storage backend by default. If you need test
results to persist across restarts, you may want to use the Postgres backend instead.

#### DNS

The DNS server is disabled by default, because its configuration options are
currently very specific to the setup over at https://log4shell.tools. Let me
know if you'd like to help make these more generic.
