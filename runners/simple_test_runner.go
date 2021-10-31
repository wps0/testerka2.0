package runners

import (
	"errors"
	"io/ioutil"
	"log"
	"math"
	"runtime"
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


type SimpleTestRunnerConfig struct {
	TestRunnerConfig
	ConcurrentRunnersAmount uint
	ConcurrentReadersAmount uint
	MaxStoreSize uint64
}

type SimpleTestRunner struct {
	Store InMemoryTestStore
	Config SimpleTestRunnerConfig

	Finished chan TestResult
	TestSizeChange chan bool
	Quit chan bool
	ReaderQuit chan TestData
}

func (runner *SimpleTestRunner) Init(cfg TestRunnerConfig) {
	runner.Config = SimpleTestRunnerConfig{
		TestRunnerConfig:        cfg,
		ConcurrentRunnersAmount: uint(math.Max(float64(runtime.NumCPU()-1), 1)),
		ConcurrentReadersAmount: 1,
		// 64 MB
		MaxStoreSize: 64*1024*1024,
	}
	runner.Finished = make(chan TestResult, runner.Config.ConcurrentRunnersAmount)
	runner.Quit = make(chan bool)
	runner.ReaderQuit = make(chan TestData, runner.Config.ConcurrentReadersAmount)
}

func (runner *SimpleTestRunner) TestReader() {
	inputFiles, err := ioutil.ReadDir(runner.Config.InputDataDir)
	if err != nil {
		log.Fatalf("Cannot read the input data direcory! Error: %v\n", err)
	}
	outputFiles, err := ioutil.ReadDir(runner.Config.OutputDataDir)
	if err != nil {
		log.Fatalf("Cannot read the output data direcory! Error: %v\n", err)
	}

	// id -> output file
	var idOutputMap = make(map[string]string)
	for _, output := range outputFiles {
		if output.IsDir() {
			continue
		}

		id := runner.Config.TestIdRegexp.Find([]byte(output.Name()))
		if id == nil {
			log.Printf("Invalid path: %s\n", output.Name())
		}
		idOutputMap[string(id)] = output.Name()
	}

	//readersAmount := uint(0)
	for _, input := range inputFiles {
		id := string(runner.Config.TestIdRegexp.Find([]byte(input.Name())))
		val, exists := idOutputMap[id]
		if !exists {
			log.Printf("Output file not found. Test path: %s\n", input.Name())
		}

		//if readersAmount < runner.Config.ConcurrentReadersAmount {
		//	// co to to input name??
		//	go runner.ReadTest(input.Name(), val)
		//	readersAmount++
		//	continue
		//}

		select {

		}
	}

}

func (runner *SimpleTestRunner) TestDealer() {
	panic("implement me")
}

func (runner *SimpleTestRunner) ReadTest(inputPath string, outputPath string) {
	
	panic("implement me")
}

func (runner *SimpleTestRunner) RunTest(test *TestData) {
	panic("implement me")
}

func (runner *SimpleTestRunner) CheckResult(data *TestData, report *TestReport) TestResult {
	panic("implement me")
}

