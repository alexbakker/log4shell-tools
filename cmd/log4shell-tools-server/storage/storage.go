package storage

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	TestResultDnsQuery   = "recv_dns_query"
	TestResultLdapBind   = "recv_ldap_bind"
	TestResultLdapSearch = "recv_ldap_search"
	TestResultHttpGet    = "recv_http_get"
)

type Backend interface {
	Close()
	Test(ctx context.Context, id uuid.UUID) (*Test, error)
	InsertTest(ctx context.Context, id uuid.UUID) error
	InsertTestResult(ctx context.Context, t *Test, resultType string, addr string, ptr *string) error
	TestResults(ctx context.Context, t *Test) ([]*TestResult, error)
	PruneTestResults(ctx context.Context) (int64, error)
	FinishTest(ctx context.Context, t *Test) error
	ActiveTests(ctx context.Context, timeout time.Duration) (int64, error)
}

type Test struct {
	ID       uuid.UUID  `db:"id"`
	Created  *time.Time `db:"created"`
	Finished *time.Time `db:"finished"`
}

type TestResult struct {
	TestID  uuid.UUID  `db:"test_id"`
	Created *time.Time `db:"created"`
	Type    string     `db:type`
	Addr    *string    `db:ip`
	Ptr     *string    `db:ptr`
}

func NewBackend(connStr string) (Backend, error) {
	if strings.HasPrefix(connStr, "memory://") {
		return NewMemory(), nil
	}

	return NewDB(connStr)
}

func (t *Test) Done(timeout time.Duration) bool {
	return t.Finished != nil || t.TimedOut(timeout)
}

func (t *Test) TimedOut(d time.Duration) bool {
	return time.Since(*t.Created) > d
}

func (r *TestResult) Color() string {
	if r.Type == TestResultHttpGet {
		return "danger"
	}

	return "warning"
}
