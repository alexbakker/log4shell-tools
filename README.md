# log4shell.tools

__log4shell.tools__ is a tool that allows you to run a test to check whether one of
your applications is affected by a vulnerability in log4j: __CVE-2021-44228__.

This is the code that runs https://log4shell.tools. If you'd like to inspect the
code or run an instance in your own environment, you've come to the right
place.

<img width="750" src="https://alexbakker.me/u/iq8qmxclfb.png"/>

## Installation

The tool was tested on Go 1.16. Make sure it (or a more recent version) is
installed and run the following command:

```
go install github.com/alexbakker/log4shell-tools/cmd/log4shell-tools-server
```

The binary will be available in ``$GOPATH/bin``

### Usage

The tool uses its in-memory storage backend by default. If you need test
results to persist across restarts, you may want to use the Postgres backend instead.

```
Usage of log4shell-tools-server:

This tool only listens on 127.0.0.1 by default. Set the addr-* options to customize for your environment.

  -addr-http string
    	listening address for the HTTP server (default "127.0.0.1:8001")
  -addr-http-external string
    	address where the HTTP server can be reached externally (default "127.0.0.1:8001")
  -addr-ldap string
    	listening address for the LDAP server (default "127.0.0.1:12345")
  -addr-ldap-external string
    	address where the LDAP server can be reached externally (default "127.0.0.1:12345")
  -http-proto string
    	the HTTP protocol to use for URL's (default "https")
  -storage string
    	storage connection URI (either memory:// or a postgres:// URI (default "memory://")
  -test-timeout int
    	test timeout in minutes (default 30)
```
