package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"

	ldap "github.com/alexbakker/ldapserver"
	"github.com/alexbakker/log4shell-tools/cmd/log4shell-tools-server/storage"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	flagStorage          = flag.String("storage", "memory://", "storage connection URI (either memory:// or a postgres:// URI")
	flagDNSEnable        = flag.Bool("dns-enable", false, "enable the DNS server")
	flagDNSAddr          = flag.String("dns-addr", "127.0.0.1:12346", "listening address for the DNS server")
	flagDNSZone          = flag.String("dns-zone", "", "DNS zone that is forwarded to the tool's DNS server (example: \"dns.log4shell.tools\")")
	flagDNSA             = flag.String("dns-a", "127.0.0.1", "the IPv4 address to respond with to any A record queries for 'dns-zone'")
	flagDNSAAAA          = flag.String("dns-aaaa", "::1", "the IPv6 address to respond with to any AAAA record queries for 'dns-zone'")
	flagLDAPAddr         = flag.String("ldap-addr", "127.0.0.1:12345", "listening address for the LDAP server")
	flagLDAPAddrExternal = flag.String("ldap-addr-external", "127.0.0.1:12345", "address where the LDAP server can be reached externally")
	flagLDAPHTTPProto    = flag.String("ldap-http-proto", "http", "the HTTP protocol to use in the payload URL that the LDAP server responds with")
	flagHTTPAddr         = flag.String("http-addr", "127.0.0.1:8001", "listening address for the HTTP server")
	flagHTTPAddrExternal = flag.String("http-addr-external", "127.0.0.1:8001", "address where the HTTP server can be reached externally")
	flagTestTimeout      = flag.Int("test-timeout", 30, "test timeout in minutes")
	testTimeout          = time.Duration(*flagTestTimeout)

	className = "Log4Shell"

	//go:embed templates/index.html
	tmplIndexText string
	tmplIndex     *template.Template

	store      storage.Backend
	statsCache *StatsCache
)

type IndexModel struct {
	New              bool
	UUID             uuid.UUID
	Test             *storage.Test
	Context          context.Context
	AddrLDAP         string
	AddrLDAPExternal string
	DNSEnabled       bool
	DNSZone          string
	ActiveTests      int64
	Error            string
}

type StatsCache struct {
	store              storage.Backend
	l                  sync.Mutex
	activeTests        int64
	activeTestsFetched time.Time
}

func init() {
	ldap.Logger = ldap.DiscardingLogger

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "This tool only listens on 127.0.0.1 by default. Pass the flags below to customize for your environment.\n\n")
		flag.PrintDefaults()
	}

	tmplFuncs := template.FuncMap{
		"IsTestDone": func(test *storage.Test) bool {
			return test.Done(testTimeout)
		},
		"IsTestTimedOut": func(test *storage.Test) bool {
			return test.TimedOut(testTimeout)
		},
		"GetTestResults": func(ctx context.Context, test *storage.Test) ([]*storage.TestResult, error) {
			return store.TestResults(ctx, test)
		},
	}

	var err error
	tmplIndex, err = template.New("index").Funcs(tmplFuncs).Parse(tmplIndexText)
	if err != nil {
		log.WithError(err).Fatal("Unable to load template")
	}
}

func main() {
	flag.Parse()
	testTimeout = time.Minute * time.Duration(*flagTestTimeout)

	log.WithField("uri", *flagStorage).Info("Opening storage backend")
	var err error
	store, err = storage.NewBackend(*flagStorage)
	if err != nil {
		log.WithError(err).Fatal("Unable to open storage backend")
	}
	defer store.Close()
	statsCache = &StatsCache{store: store}

	go func() {
		for {
			<-time.After(1 * time.Minute)

			deleted, err := store.PruneTestResults(context.Background())
			if err != nil {
				log.WithError(err).Error("Unable to delete old test results")
			} else {
				log.WithField("count", deleted).Info("Deleted old test results")
			}
		}
	}()

	ldapServer := NewLDAPServer(store)
	go func() {
		log.WithFields(log.Fields{
			"addr":     *flagLDAPAddr,
			"addr_ext": *flagLDAPAddrExternal,
		}).Info("Starting LDAP server")

		if err := ldapServer.ListenAndServe(*flagLDAPAddr); err != nil {
			log.WithError(err).Fatal("Unable to run LDAP server")
		}
	}()

	if *flagDNSEnable {
		dnsServer := NewDNSServer(store, DNSServerOpts{
			Addr: *flagDNSAddr,
			Zone: *flagDNSZone,
			A:    *flagDNSA,
			AAAA: *flagDNSAAAA,
		})

		go func() {
			log.WithFields(log.Fields{
				"addr": *flagDNSAddr,
			}).Info("Starting DNS server")

			if err := dnsServer.ListenAndServe(); err != nil {
				log.WithError(err).Fatal("Unable to run DNS server")
			}
		}()
	}

	promHandler := promhttp.Handler()
	router := httprouter.New()
	router.GET("/", handleIndex)
	router.GET("/metrics", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		promHandler.ServeHTTP(w, r)
	})
	router.GET(fmt.Sprintf("/api/tests/:uuid/payload/%s.class", className), handleTestPayloadDownload)

	log.WithFields(log.Fields{
		"addr":     *flagHTTPAddr,
		"addr_ext": *flagHTTPAddrExternal,
	}).Info("Starting HTTP server")

	if err := http.ListenAndServe(*flagHTTPAddr, router); err != nil {
		log.WithError(err).Fatal("Unable to start HTTP server")
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	model := IndexModel{
		Context:          r.Context(),
		ActiveTests:      statsCache.getActiveTests(r.Context()),
		AddrLDAP:         *flagLDAPAddr,
		AddrLDAPExternal: *flagLDAPAddrExternal,
		DNSEnabled:       *flagDNSEnable,
		DNSZone:          *flagDNSZone,
	}
	ctxLog := log.WithFields(log.Fields{
		"server": "http",
		"addr":   getRemoteAddr(r),
		"req":    r.URL.Path,
	})

	idString := r.URL.Query().Get("uuid")
	if idString != "" {
		var err error
		if model.UUID, err = uuid.Parse(idString); err != nil {
			ctxLog.WithField("id", idString).WithError(err).Error("Unable to parse UUID")
			writeHttpError(w, http.StatusBadRequest)
			return
		}
		ctxLog = ctxLog.WithField("test", model.UUID)

		model.Test, err = store.Test(r.Context(), model.UUID)
		if err != nil {
			ctxLog.WithError(err).Error("Unable to lookup test in storage")
			writeHttpError(w, http.StatusInternalServerError)
			return
		}
		if model.Test == nil {
			if r.URL.Query().Get("terms") != "y" {
				model.Error = "You cannot continue before agreeing to only testing on machines that you have permission to test on."
			} else {
				ctxLog.Info("Inserting new test")

				if err := store.InsertTest(r.Context(), model.UUID); err != nil {
					ctxLog.WithError(err).Error("Unable to insert new test")
					writeHttpError(w, http.StatusInternalServerError)
					return
				}
				if model.Test, err = store.Test(r.Context(), model.UUID); err != nil {
					ctxLog.WithError(err).Error("Unable to lookup test in storage")
					writeHttpError(w, http.StatusInternalServerError)
					return
				}
				if model.Test == nil {
					ctxLog.Error("Unable to obtain test right after insert")
					writeHttpError(w, http.StatusInternalServerError)
					return
				}

				counterTestsCreated.Inc()
			}
		}
	} else {
		model.New = true
		model.UUID = uuid.New()
	}

	var buf bytes.Buffer
	if err := tmplIndex.Execute(&buf, model); err != nil {
		ctxLog.WithError(err).Error("Unable to render template")
		writeHttpError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "text/html")
	w.Write(buf.Bytes())
}

func handleTestPayloadDownload(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	ctxLog := log.WithFields(log.Fields{
		"server": "http",
		"addr":   getRemoteAddr(r),
		"req":    r.URL.Path,
	})

	idString := p.ByName("uuid")
	id, err := uuid.Parse(idString)
	if err != nil {
		ctxLog.WithField("id", idString).WithError(err).Error("Unable to parse UUID")
		writeHttpError(w, http.StatusBadRequest)
		return
	}

	ctxLog = ctxLog.WithField("test", id)
	ctxLog.Info("Handling payload download request")
	counterPayloadRequests.Inc()

	test, err := store.Test(r.Context(), id)
	if err != nil {
		ctxLog.WithError(err).Error("Unable to lookup test in storage")
		writeHttpError(w, http.StatusInternalServerError)
		return
	}
	if test == nil {
		ctxLog.Warn("Test not found")
		writeHttpError(w, http.StatusNotFound)
		return
	}
	if test.Done(testTimeout) {
		ctxLog.Warn("Test already done")
		writeHttpError(w, http.StatusGone)
		return
	}

	addr, ptr := getAddrPtr(r.Context(), getRemoteAddr(r))
	if err = store.InsertTestResult(r.Context(), test, storage.TestResultHttpGet, addr, ptr); err != nil {
		ctxLog.WithError(err).Error("Unable to insert test result")
		writeHttpError(w, http.StatusInternalServerError)
		return
	}
	if err = store.FinishTest(r.Context(), test); err != nil {
		ctxLog.WithError(err).Error("Unable to mark test as finished")
		writeHttpError(w, http.StatusInternalServerError)
		return
	}
	counterTestsCompleted.Inc()

	writeHttpError(w, http.StatusNotFound)
}

func writeHttpError(w http.ResponseWriter, code int) {
	http.Error(w, fmt.Sprintf("%d - %s", code, http.StatusText(code)), code)
}

func (c *StatsCache) getActiveTests(ctx context.Context) int64 {
	c.l.Lock()
	defer c.l.Unlock()

	if time.Since(c.activeTestsFetched) > 1*time.Minute {
		var err error
		c.activeTests, err = c.store.ActiveTests(ctx, testTimeout)
		if err != nil {
			log.WithError(err).Error("Unable to fetch number of active tests")
		}
		c.activeTestsFetched = time.Now()
	}

	return c.activeTests
}
