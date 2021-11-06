package runners

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"os/exec"
	"runtime"
	"sort"
	"sync"
)

type XORedResult struct {
	StartN int64
	Xor    uint64
}

type SingleFileTestRunnerConfig struct {
	TestRunnerConfig
	ConcurrentRunnersAmount uint
	PrintingModulo          uint
	XorDefaultValue         uint
	UpToN                   int64
}

type SingleFileTestRunner struct {
	Config    SingleFileTestRunnerConfig
	WaitGroup sync.WaitGroup

	Stats    TestStats
	StatsMux sync.RWMutex

	TestRequests chan int64
	Results      chan XORedResult

	Quit chan bool
}

func (runner *SingleFileTestRunner) GetStats() TestStats {
	runner.StatsMux.RLock()
	defer runner.StatsMux.RUnlock()
	return runner.Stats
}

func (runner *SingleFileTestRunner) Init(cfg TestRunnerConfig) {
	runner.Quit = make(chan bool)
	cpuCount := runtime.NumCPU()
	runner.Config = SingleFileTestRunnerConfig{
		TestRunnerConfig:        cfg,
		ConcurrentRunnersAmount: uint(math.Max(float64(cpuCount)*1.25, 1)),
		PrintingModulo:          100000,
		XorDefaultValue:         0,
		UpToN:                   1000000000,
	}
	runner.TestRequests = make(chan int64, runner.Config.ConcurrentRunnersAmount*2)
	runner.Results = make(chan XORedResult, runner.Config.ConcurrentRunnersAmount*1)
	runner.InitTestRunners()
	go runner.TestRequestsSupplier()
	go runner.TestResultsCombiner()
}

func (runner *SingleFileTestRunner) TestRequestsSupplier() {
	runner.WaitGroup.Add(1)
	defer runner.WaitGroup.Done()
	for i := int64(runner.Config.SkipFirstN) + 1; i <= runner.Config.UpToN; i += 1000 {
		runner.TestRequests <- i
	}
	runner.TestRequests <- -1
}

func (runner *SingleFileTestRunner) TestReader() {
	panic("stub method")
}

func (runner *SingleFileTestRunner) InitTestRunners() {
	log.Printf("Running %d test runners...\n", runner.Config.ConcurrentRunnersAmount)
	for i := uint(0); i < runner.Config.ConcurrentRunnersAmount; i++ {
		go runner.TestRunner()
	}
}

func (runner *SingleFileTestRunner) TestResultsCombiner() {
	runner.WaitGroup.Add(1)
	defer runner.WaitGroup.Done()
	cache := map[int64]uint64{}
	fullXor := uint64(0)
	nextStartN := int64(1)
	for {
		select {
		case res := <-runner.Results:
			cache[res.StartN] = res.Xor
			if _, exists := cache[nextStartN]; exists {
				keys := make([]int64, 0, len(cache))
				for key, _ := range cache {
					keys = append(keys, key)
				}
				sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
				for _, key := range keys {
					if key != nextStartN {
						break
					}
					fullXor ^= cache[key]
					delete(cache, key)
					nextStartN += 1000
				}
			}
		case <-runner.Quit:
			return
		}
	}
}

func (runner *SingleFileTestRunner) TestRunner() {
	runner.WaitGroup.Add(1)
	defer runner.WaitGroup.Done()
	for {
		select {
		case startI, ok := <-runner.TestRequests:
			if !ok {
				return
			}
			if startI == -1 {
				close(runner.TestRequests)
				close(runner.Quit)
				return
			}

			xor := uint64(0)
			for i := startI; i < startI+1000; i++ {
				inData := make([]byte, 8)
				binary.LittleEndian.PutUint64(inData, uint64(i))
				report := runner.RunTest(&TestData{InputData: inData})
				xor ^= binary.LittleEndian.Uint64(report.Output)
			}
			runner.Results <- XORedResult{
				StartN: startI,
				Xor:    xor,
			}
		case <-runner.Quit:
			return
		}
	}
}

func (runner *SingleFileTestRunner) ReadTest(test TestLocation) TestData {
	panic("stub method")
}

func (runner *SingleFileTestRunner) RunTest(test *TestData) TestReport {
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

func (runner *SingleFileTestRunner) CheckResult(data *TestData, report *TestReport) TestResult {
	panic("stub method")
}
