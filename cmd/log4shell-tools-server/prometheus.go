package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	counterTestsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "log4shell_tests_created_total",
		Help: "The total number of new tests inserted into the database",
	})
	counterTestsCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "log4shell_tests_completed_total",
		Help: "The total number of new tests that were completed (no timeout)",
	})
	counterBindRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "log4shell_ldap_bind_requests_total",
		Help: "The total number of LDAP bind requests",
	})
	counterSearchRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "log4shell_ldap_search_requests_total",
		Help: "The total number of LDAP search requests",
	})
	counterPayloadRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "log4shell_http_payload_requests_total",
		Help: "The total number of RCE payload requests",
	})
	counterDNSQueries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "log4shell_dns_queries_total",
		Help: "The total number of DNS queries",
	})
)
