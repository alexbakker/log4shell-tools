package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type testWrapper struct {
	Test    *Test
	Results []*TestResult
}

type Memory struct {
	tests map[uuid.UUID]*testWrapper
	l     sync.Mutex
}

func NewMemory() *Memory {
	return &Memory{tests: map[uuid.UUID]*testWrapper{}}
}

func (m *Memory) Close() {

}

func (m *Memory) Test(ctx context.Context, id uuid.UUID) (*Test, error) {
	m.l.Lock()
	defer m.l.Unlock()

	tw, ok := m.tests[id]
	if !ok {
		return nil, nil
	}

	copy := *tw.Test
	return &copy, nil
}

func (m *Memory) InsertTest(ctx context.Context, id uuid.UUID) error {
	m.l.Lock()
	defer m.l.Unlock()

	_, ok := m.tests[id]
	if ok {
		return fmt.Errorf("not found: %s", id)
	}

	now := time.Now().UTC()
	m.tests[id] = &testWrapper{Test: &Test{ID: id, Created: &now}}
	return nil
}

func (m *Memory) InsertTestResult(ctx context.Context, t *Test, resultType string, addr string, ptr *string) error {
	m.l.Lock()
	defer m.l.Unlock()

	tw, ok := m.tests[t.ID]
	if !ok {
		return fmt.Errorf("not found: %s", t.ID)
	}

	now := time.Now().UTC()
	tw.Results = append(tw.Results, &TestResult{
		TestID:  tw.Test.ID,
		Created: &now,
		Type:    resultType,
		Addr:    &addr,
		Ptr:     ptr,
	})
	return nil
}

func (m *Memory) TestResults(ctx context.Context, t *Test) ([]*TestResult, error) {
	m.l.Lock()
	defer m.l.Unlock()

	tw, ok := m.tests[t.ID]
	if !ok {
		return nil, fmt.Errorf("not found: %s", t.ID)
	}

	var res []*TestResult
	for _, tres := range tw.Results {
		copy := *tres
		res = append(res, &copy)
	}

	return res, nil
}

func (m *Memory) PruneTestResults(ctx context.Context) (int64, error) {
	m.l.Lock()
	defer m.l.Unlock()

	var count int64
	for id, tw := range m.tests {
		if time.Since(*tw.Test.Created) > time.Hour*24 {
			delete(m.tests, id)
			count++
		}
	}

	return count, nil
}

func (m *Memory) FinishTest(ctx context.Context, t *Test) error {
	m.l.Lock()
	defer m.l.Unlock()

	tw, ok := m.tests[t.ID]
	if !ok {
		return fmt.Errorf("not found: %s", t.ID)
	}

	now := time.Now().UTC()
	tw.Test.Finished = &now
	return nil
}

func (m *Memory) ActiveTests(ctx context.Context, timeout time.Duration) (int64, error) {
	m.l.Lock()
	defer m.l.Unlock()

	var count int64
	for _, tw := range m.tests {
		if tw.Test.Finished == nil && time.Since(*tw.Test.Created) < timeout {
			count++
		}
	}

	return count, nil
}
