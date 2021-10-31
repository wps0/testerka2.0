package runners

import (
	"errors"
	"sync"
)

type InMemoryTestStore struct {
	Tests map[string]TestData
	// in bytes
	TestsSize uint64
	Mux sync.RWMutex
}

func (store *InMemoryTestStore) Get(id string) (TestData, error) {
	store.Mux.RLock()
	defer store.Mux.RUnlock()

	val, exists := store.Tests[id]
	if exists {
		return val, nil
	}
	return TestData{}, errors.New("key does not exists")
}

func (store *InMemoryTestStore) Exists(id string) bool {
	store.Mux.RLock()
	defer store.Mux.RUnlock()

	_, exists := store.Tests[id]
	return exists
}

func (store *InMemoryTestStore) Insert(id string, data TestData) error {
	if store.Exists(id) {
		return errors.New("id " + id + " is not unique (has been already used)")
	}
	store.Mux.Lock()
	defer store.Mux.Unlock()

	store.Tests[id] = data
	store.TestsSize += uint64(len(data.ExpectedOutput)) + uint64(len(data.InputData))
	return nil
}

func (store *InMemoryTestStore) Remove(id string) error {
	if store.Exists(id) {
		store.Mux.Lock()
		defer store.Mux.Unlock()

		store.TestsSize -= uint64(len(store.Tests[id].ExpectedOutput)) + uint64(len(store.Tests[id].InputData))
		delete(store.Tests, id)
		return nil
	}
	return errors.New("cannot delete a nonexistent key")
}

func (store *InMemoryTestStore) Size() uint64 {
	store.Mux.RLock()
	defer store.Mux.RUnlock()

	return store.TestsSize
}


type SimpleTestRunner struct {
	Store InMemoryTestStore
}

func (runner *SimpleTestRunner) Init(cfg TestRunnerConfiguration) {
	panic("implement me")
}

func (runner *SimpleTestRunner) TestReader(store *TestStore) {
	panic("implement me")
}

func (runner *SimpleTestRunner) TestDealer(store *TestStore) {
	panic("implement me")
}

func (runner *SimpleTestRunner) ReadTest() TestData {
	panic("implement me")
}

func (runner *SimpleTestRunner) RunTest(test *TestData) {
	panic("implement me")
}

func (runner *SimpleTestRunner) CheckResult(data *TestData, report *TestReport) TestResult {
	panic("implement me")
}

