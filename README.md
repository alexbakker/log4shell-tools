# log4shell-tools

This tool allows you to run a test to check whether one of your machines is
affected by a vulnerability in log4j: CVE-2021-44228.

## Usage

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
