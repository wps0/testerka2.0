package runners

import (
	"errors"
	"io/ioutil"
	"log"
	"math"
	"os"
	"runtime"
	"sync"
)

type InMemoryTestStore struct {
	Tests map[string]TestData
	// in bytes
	TestsSize uint64
	Mux sync.RWMutex
	OnSizeChange func()
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
		panic("XD")
		return errors.New("id " + id + " is not unique (has been already used)")
	}
	store.Mux.Lock()
	defer store.Mux.Unlock()

	store.Tests[id] = data
	store.TestsSize += uint64(len(data.ExpectedOutput)) + uint64(len(data.InputData))
	go store.OnSizeChange()

	return nil
}

func (store *InMemoryTestStore) Remove(id string) error {
	if store.Exists(id) {
		store.Mux.Lock()
		defer store.Mux.Unlock()

		delete(store.Tests, id)
		store.TestsSize -= uint64(len(store.Tests[id].ExpectedOutput)) + uint64(len(store.Tests[id].InputData))
		go store.OnSizeChange()

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

type ReadRequest struct {
	TestLocation
	TestId string
}

type SimpleTestRunner struct {
	Store InMemoryTestStore
	Config SimpleTestRunnerConfig
	WaitGroup sync.WaitGroup

	ReadRequests chan ReadRequest
	ReadFinished chan TestData

	Finished chan TestResult
	TestSizeChange chan bool
	Quit chan bool
}

func (runner *SimpleTestRunner) Init(cfg TestRunnerConfig) {
	runner.TestSizeChange = make(chan bool)
	runner.Quit = make(chan bool)

	runner.Config = SimpleTestRunnerConfig{
		TestRunnerConfig:        cfg,
		ConcurrentRunnersAmount: uint(math.Max(float64(runtime.NumCPU()-1), 1)),
		ConcurrentReadersAmount: 1,
		// 64 MB
		MaxStoreSize: 64*1024*1024,
	}
	runner.Store = InMemoryTestStore{
		Tests:     make(map[string]TestData),
		TestsSize: 0,
		Mux:       sync.RWMutex{},
		OnSizeChange: func() {
			runner.TestSizeChange <- true
			log.Printf("new size: %d\n", runner.Store.Size())
		},
	}

	runner.ReadRequests = make(chan ReadRequest, runner.Config.ConcurrentReadersAmount)
	runner.ReadFinished = make(chan TestData, runner.Config.ConcurrentReadersAmount)

	runner.InitReaders()
	go runner.ReadersSupervisor()

	runner.Finished = make(chan TestResult, runner.Config.ConcurrentRunnersAmount)
}

func (runner *SimpleTestRunner) InitReaders() {
	log.Printf("Running %d readers...\n", runner.Config.ConcurrentReadersAmount)
	for i := uint(0); i < runner.Config.ConcurrentRunnersAmount; i++ {
		go runner.TestReader()
	}
}

func (runner *SimpleTestRunner) ReadersSupervisor()  {
	runner.WaitGroup.Add(1)
	defer runner.WaitGroup.Done()
	defer close(runner.ReadRequests)
	defer func() {
		runner.Quit <- true
	}()

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
		idOutputMap[string(id)] = runner.Config.OutputDataDir + string(os.PathSeparator) + output.Name()
	}

	for _, input := range inputFiles {
		id := string(runner.Config.TestIdRegexp.Find([]byte(input.Name())))
		outputPath, exists := idOutputMap[id]
		if !exists {
			log.Printf("Output file not found. Input path: %s\n", input.Name())
			continue
		}

		runner.ReadRequests <- ReadRequest{
			TestId: id,
			TestLocation: TestLocation{
				InputFilePath:  runner.Config.InputDataDir + string(os.PathSeparator) + input.Name(),
				OutputFilePath: outputPath,
			},
		}
	}
}

func (runner *SimpleTestRunner) TestReader() {
	runner.WaitGroup.Add(1)
	defer runner.WaitGroup.Done()
	for {
		select {
		case rq, ok := <- runner.ReadRequests:
			if !ok {
				return
			}
			log.Printf("Reading test %s\n", rq.TestId)
			testData := runner.ReadTest(rq.TestLocation)

			if runner.Config.MaxStoreSize < runner.Store.Size() {
				for {
					select {
					case <- runner.TestSizeChange:
						if runner.Config.MaxStoreSize < runner.Store.Size() {
							break
						}
					case <- runner.Quit:
						log.Printf("Test reader is quitting...")
						return
					}
				}
			}

			err := runner.Store.Insert(rq.TestId, testData)
			if err != nil {
				log.Printf("Failed to insert test to store. Error: %v\n", err)
			}
		case <- runner.Quit:
			log.Printf("Test reader is quitting...")
			return
		}
	}
}

func (runner *SimpleTestRunner) TestDealer() {
	panic("implement me")
}

func (runner *SimpleTestRunner) ReadTest(test TestLocation) TestData {
	log.Printf("wtf reading %s\n", test.InputFilePath)
	inF, err := os.Open(test.InputFilePath)
	defer inF.Close()
	if err != nil {
		log.Printf("Failed to open file %s. Error: %v\n", test.InputFilePath, err)
		return TestData{}
	}
	testData := TestData{}
	_, err = inF.Read(testData.InputData)
	if err != nil {
		log.Printf("Failed to read from file file %s. Error: %v\n", test.InputFilePath, err)
		return TestData{}
	}

	outF, err := os.Open(test.OutputFilePath)
	defer outF.Close()
	if err != nil {
		log.Printf("Failed to open file %s. Error: %v\n", test.OutputFilePath, err)
		return TestData{}
	}
	_, err = outF.Read(testData.ExpectedOutput)
	if err != nil {
		log.Printf("Failed to read from file file %s. Error: %v\n", test.OutputFilePath, err)
		return TestData{}
	}
	return testData
}

func (runner *SimpleTestRunner) RunTest(test *TestData) {
	panic("implement me")
}

func (runner *SimpleTestRunner) CheckResult(data *TestData, report *TestReport) TestResult {
	panic("implement me")
}

