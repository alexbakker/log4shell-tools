package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/alexbakker/log4shell-tools/cmd/log4shell-tools-server/storage"
	"github.com/google/uuid"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

type DNSServer struct {
	s     *dns.Server
	store storage.Backend
	opts  DNSServerOpts
}

type DNSServerOpts struct {
	Addr string
	Zone string
	A    string
	AAAA string
}

func NewDNSServer(store storage.Backend, opts DNSServerOpts) *DNSServer {
	s := DNSServer{store: store, opts: opts}
	mux := dns.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("%s.", opts.Zone), s.handleDNSQuery)
	server := &dns.Server{Addr: opts.Addr, Net: "udp", Handler: mux}
	s.s = server
	return &s
}

func (s *DNSServer) ListenAndServe() error {
	return s.s.ListenAndServe()
}

func (s *DNSServer) handleDNSQuery(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	if r.Opcode == dns.OpcodeQuery {
		if len(m.Question) == 0 {
			w.WriteMsg(m)
			return
		}

		q := m.Question[0]
		if q.Qtype != dns.TypeA && q.Qtype != dns.TypeAAAA {
			m.SetRcode(r, dns.RcodeSuccess)
			w.WriteMsg(m)
			return
		}

		if strings.HasPrefix(strings.ToLower(q.Name), s.opts.Zone) {
			s.writeDNSRes(m, r, q.Name, q.Qtype)
			w.WriteMsg(m)
			return
		}

		ctxLog := log.WithFields(log.Fields{
			"server": "dns",
			"addr":   w.RemoteAddr().String(),
			"q":      q.Name,
			"type":   dns.TypeToString[q.Qtype],
		})

		parts := strings.Split(q.Name, ".")
		id, err := uuid.Parse(parts[0])
		if err != nil {
			ctxLog.WithError(err).Error("Unable to parse UUID")
			m.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(m)
			return
		}

		ctxLog = ctxLog.WithField("test", id)
		ctxLog.Info("Handling DNS query")
		counterDNSQueries.Inc()

		test, err := s.store.Test(context.Background(), id)
		if err != nil {
			ctxLog.WithError(err).Error("Unable to lookup test in storage")
			m.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(m)
			return
		}
		if test == nil {
			ctxLog.Warn("Test not found")
			m.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(m)
			return
		}
		if test.Done(testTimeout) {
			ctxLog.Warn("Test already done")
			m.SetRcode(r, dns.RcodeNameError)
			w.WriteMsg(m)
			return
		}

		addr, ptr := getAddrPtr(context.Background(), w.RemoteAddr().String())
		if err = s.store.InsertTestResult(context.Background(), test, storage.TestResultDnsQuery, addr, ptr); err != nil {
			ctxLog.WithError(err).Error("Unable to insert test result")
			w.WriteMsg(m)
			return
		}

		s.writeDNSRes(m, r, q.Name, q.Qtype)
	}

	w.WriteMsg(m)
}

func (s *DNSServer) writeDNSRes(m *dns.Msg, r *dns.Msg, name string, recordType uint16) {
	var record string
	switch recordType {
	case dns.TypeA:
		record = s.opts.A
	case dns.TypeAAAA:
		record = s.opts.AAAA
	}

	if record != "" {
		rr, err := dns.NewRR(fmt.Sprintf("%s %s %s", name, dns.TypeToString[recordType], record))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		}
	}
}
