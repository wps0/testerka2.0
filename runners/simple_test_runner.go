package runners

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

type InMemoryTestStore struct {
	Tests map[string]TestData
	// in bytes
	TestsSize    uint64
	Mux          sync.RWMutex
	OnSizeChange func()
}

type Test struct {
	Id   string
	Data TestData
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
	MaxTestCache            uint64
}

type ReadRequest struct {
	TestLocation
	TestId string
}

type SimpleTestRunner struct {
	//Store     InMemoryTestStore
	Config    SimpleTestRunnerConfig
	WaitGroup sync.WaitGroup

	Stats    TestStats
	StatsMux sync.RWMutex

	ReadRequests chan ReadRequest
	ReadyTests   chan Test

	Finished       chan TestResult
	TestSizeChange chan bool
	Quit           chan bool
}

func (runner *SimpleTestRunner) GetStats() TestStats {
	runner.StatsMux.RLock()
	defer runner.StatsMux.RUnlock()
	return runner.Stats
}

func (runner *SimpleTestRunner) Init(cfg TestRunnerConfig) {
	runner.TestSizeChange = make(chan bool)
	runner.Quit = make(chan bool)

	cpuCount := runtime.NumCPU()

	runner.Config = SimpleTestRunnerConfig{
		TestRunnerConfig:        cfg,
		ConcurrentRunnersAmount: uint(math.Max(float64(cpuCount)-1, 1) * 1.25),
		ConcurrentReadersAmount: uint(math.Floor(float64(cpuCount) * 1.15)),
		MaxTestCache:            1024,
	}

	runner.ReadRequests = make(chan ReadRequest, runner.Config.ConcurrentReadersAmount)
	runner.ReadyTests = make(chan Test, runner.Config.MaxTestCache)

	runner.InitReaders()
	runner.WaitGroup.Add(1)
	go runner.ReadersSupervisor()
	runner.InitTestRunners()

	runner.Finished = make(chan TestResult, runner.Config.ConcurrentRunnersAmount)
}

func (runner *SimpleTestRunner) InitReaders() {
	log.Printf("Running %d readers...\n", runner.Config.ConcurrentReadersAmount)
	for i := uint(0); i < runner.Config.ConcurrentReadersAmount; i++ {
		go runner.TestReader()
	}
}

func (runner *SimpleTestRunner) ReadersSupervisor() {
	defer runner.WaitGroup.Done()

	log.Println("Reading the output directory...")
	outputFiles, err := ioutil.ReadDir(runner.Config.OutputData)
	if err != nil {
		log.Fatalf("Cannot read the output data direcory! Error: %v\n", err)
	}

	log.Println("Assigning each output file a unique id...")
	// id -> output file
	var idOutputMap = make(map[string]string)
	for _, output := range outputFiles {
		if output.IsDir() {
			continue
		}

		id := runner.Config.TestIdRegexpInternal.Find([]byte(output.Name()))
		if id == nil {
			log.Printf("Invalid path: %s\n", output.Name())
		}
		idOutputMap[string(id)] = runner.Config.OutputData + string(os.PathSeparator) + output.Name()
	}

	log.Println("Reading the input directory...")
	inputFiles, err := ioutil.ReadDir(runner.Config.InputData)
	if err != nil {
		log.Fatalf("Cannot read the input data direcory! Error: %v\n", err)
	}

	log.Println("Preparing the input files to be read...")
	log.Printf("Skipping first %d tests...", runner.Config.SkipFirstN)
	for i, input := range inputFiles {
		if i < int(runner.Config.SkipFirstN) {
			continue
		}
		id := string(runner.Config.TestIdRegexpInternal.Find([]byte(input.Name())))
		outputPath, exists := idOutputMap[id]
		if !exists {
			log.Printf("Output file not found. Input path: %s\n", input.Name())
			continue
		}

		runner.ReadRequests <- ReadRequest{
			TestId: id,
			TestLocation: TestLocation{
				InputFilePath:  runner.Config.InputData + string(os.PathSeparator) + input.Name(),
				OutputFilePath: outputPath,
			},
		}
	}
	runner.ReadRequests <- ReadRequest{TestId: "FORWARD-CLOSE-QUIT-CHANNEL"}
}

func (runner *SimpleTestRunner) TestReader() {
	runner.WaitGroup.Add(1)
	defer runner.WaitGroup.Done()
	for {
		select {
		case rq, ok := <-runner.ReadRequests:
			if !ok {
				return
			}
			if rq.TestId == "FORWARD-CLOSE-QUIT-CHANNEL" {
				close(runner.ReadRequests)
				runner.ReadyTests <- Test{Id: "CLOSE-QUIT-CHANNEL"}
				return
			}

			testData := runner.ReadTest(rq.TestLocation)
			runner.ReadyTests <- Test{
				Id:   rq.TestId,
				Data: testData,
			}
		case <-runner.Quit:
			log.Printf("Test reader is quitting...")
			return
		}
	}
}

func (runner *SimpleTestRunner) InitTestRunners() {
	log.Printf("Running %d test runners...\n", runner.Config.ConcurrentRunnersAmount)
	for i := uint(0); i < runner.Config.ConcurrentRunnersAmount; i++ {
		go runner.TestRunner()
	}
}

func (runner *SimpleTestRunner) TestRunner() {
	runner.WaitGroup.Add(1)
	defer runner.WaitGroup.Done()
	for {
		select {
		case test, ok := <-runner.ReadyTests:
			if !ok {
				return
			}
			if test.Id == "CLOSE-QUIT-CHANNEL" {
				close(runner.ReadyTests)
				close(runner.Quit)
				return
			}

			report := runner.RunTest(&test.Data)
			result := runner.CheckResult(&test.Data, &report)

			if result.Status {
				cnt := 0
				runner.StatsMux.Lock()
				runner.Stats.OKCount++
				cnt = runner.Stats.OKCount + runner.Stats.WACount
				runner.StatsMux.Unlock()
				if cnt%10000 == 0 {
					log.Printf("Status: %d / ?", cnt)
				}
			} else {
				runner.StatsMux.Lock()
				stats := runner.Stats
				runner.Stats.WACount++
				runner.StatsMux.Unlock()
				log.Printf("Test %s (OK: %d, WAs: %d) --  %d  %s\n", test.Id, stats.OKCount, stats.WACount, report.Time, result.Message)
			}
		case <-runner.Quit:
			return
		}
	}
}

func (runner *SimpleTestRunner) ReadTest(test TestLocation) TestData {
	inF, err := os.Open(test.InputFilePath)
	defer inF.Close()
	if err != nil {
		log.Printf("Failed to open file %s. Error: %v\n", test.InputFilePath, err)
		return TestData{}
	}
	stat, err := inF.Stat()
	if err != nil {
		log.Println(err)
		return TestData{}
	}

	testData := TestData{
		InputData: make([]byte, stat.Size()),
	}
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
	stat, err = outF.Stat()
	if err != nil {
		log.Println(err)
		return TestData{}
	}

	testData.ExpectedOutput = make([]byte, stat.Size())
	_, err = outF.Read(testData.ExpectedOutput)
	if err != nil {
		log.Printf("Failed to read from file file %s. Error: %v\n", test.OutputFilePath, err)
		return TestData{}
	}
	return testData
}

func (runner *SimpleTestRunner) RunTest(test *TestData) TestReport {
	data := TestReport{
		Time:      0,
		Message:   "OK",
		MaxMemory: 0,
		ExitCode:  0,
	}
	cmd := exec.Command(runner.Config.SolutionPath)
	cmd.Stdin = bytes.NewReader(test.InputData)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return TestReport{
			ExitCode: -1,
			Message:  err.Error(),
		}
	}
	data.Output = out
	return data
}

func (runner *SimpleTestRunner) CheckResult(data *TestData, report *TestReport) TestResult {
	var trimmedExp []byte
	var trimmedReal []byte
	if report.ExitCode == -1 {
		return TestResult{
			Status:  false,
			Message: "WA: " + report.Message,
		}
	}

	for i := len(data.ExpectedOutput) - 1; i >= 0; i-- {
		if data.ExpectedOutput[i] != '\n' && data.ExpectedOutput[i] != ' ' && data.ExpectedOutput[i] != '\r' {
			trimmedExp = data.ExpectedOutput[:i+1]
			break
		}
	}
	for i := len(report.Output) - 1; i >= 0; i-- {
		if report.Output[i] != '\n' && report.Output[i] != ' ' && report.Output[i] != '\r' {
			trimmedReal = report.Output[:i+1]
			break
		}
	}
	if bytes.Compare(trimmedExp, trimmedReal) == 0 {
		return TestResult{
			Status:  true,
			Message: "OK",
		}
	}
	return TestResult{
		Status:  false,
		Message: "WA",
	}
}
