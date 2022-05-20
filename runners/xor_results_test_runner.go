package runners

import (
	"bytes"
	"log"
	"math"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"sync"
)

type XORedResult struct {
	StartN int64
	Xor    uint64
}

type XorTestRunnerSpecificConfig struct {
	ConcurrentRunnersAmount uint
	PrintingModulo          uint64
	XorDefaultValue         uint
	UpToN                   int64
}

type XorTestRunnerConfig struct {
	GenericConfig  TestRunnerConfig
	SpecificConfig XorTestRunnerSpecificConfig
}

type XorTestRunner struct {
	Config    XorTestRunnerConfig
	WaitGroup sync.WaitGroup

	TestRequests chan int64
	Results      chan XORedResult

	Quit chan bool
}

func (runner *XorTestRunner) GetStats() TestStats {
	panic("stub method")
}

func (runner *XorTestRunner) GetWaitGroup() *sync.WaitGroup {
	return &runner.WaitGroup
}

func (runner *XorTestRunner) Init(cfg TestRunnerConfig) {
	runner.Quit = make(chan bool)
	cpuCount := runtime.NumCPU()
	runner.Config = XorTestRunnerConfig{
		GenericConfig: cfg,
		SpecificConfig: XorTestRunnerSpecificConfig{
			ConcurrentRunnersAmount: uint(math.Max(float64(cpuCount)*0.5, 1)),
			PrintingModulo:          10,
			XorDefaultValue:         0,
			UpToN:                   1000000000,
		},
	}
	runner.TestRequests = make(chan int64, runner.Config.SpecificConfig.ConcurrentRunnersAmount*2)
	runner.Results = make(chan XORedResult, runner.Config.SpecificConfig.ConcurrentRunnersAmount*1)
	runner.InitTestRunners()

	runner.WaitGroup.Add(2)
	go runner.TestRequestsSupplier()
	go runner.TestResultsCombiner()
}

func (runner *XorTestRunner) TestRequestsSupplier() {
	defer runner.WaitGroup.Done()
	for i := int64(0) + 1; i <= runner.Config.SpecificConfig.UpToN; i += 1000 {
		runner.TestRequests <- i
		log.Println(i)
	}
	runner.TestRequests <- -1
}

func (runner *XorTestRunner) TestReader() {
	panic("stub method")
}

func (runner *XorTestRunner) InitTestRunners() {
	log.Printf("Running %d test runners...\n", runner.Config.SpecificConfig.ConcurrentRunnersAmount)
	for i := uint(0); i < runner.Config.SpecificConfig.ConcurrentRunnersAmount; i++ {
		go runner.TestRunner()
	}
}

func (runner *XorTestRunner) TestResultsCombiner() {
	defer runner.WaitGroup.Done()
	cache := map[int64]uint64{}
	fullXor := uint64(0)
	nextStartN := int64(1)
	for {
		select {
		case res := <-runner.Results:
			cache[res.StartN] = res.Xor
			log.Printf("start=%d; nextStartN=%d", res.StartN, nextStartN)
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

					//if (nextStartN-1) % int64(runner.Config.SpecificConfig.PrintingModulo) == 0 {
					log.Printf("...%d  xor  = %d\n", nextStartN-1, fullXor)
					//}
					nextStartN += 1000
				}
			}
		case <-runner.Quit:
			log.Printf("xor up to %d = %d", nextStartN-1000, fullXor)
			return
		}
	}
}

func (runner *XorTestRunner) TestRunner() {
	runner.WaitGroup.Add(1)
	defer runner.WaitGroup.Done()
	for {
		select {
		case startI, ok := <-runner.TestRequests:
			if !ok {
				log.Println("test runners channel not ok")
				return
			}
			if startI == -1 {
				close(runner.TestRequests)
				close(runner.Quit)
				return
			}

			xor := uint64(0)
			for i := startI; i < startI+1000; i++ {
				input := strconv.FormatInt(i, 10)
				report := runner.RunTest(&TestData{InputData: []byte(input)})
				output := WhitespaceTrimRight(string(report.Output))
				inted, err := strconv.ParseUint(output, 10, 64)
				if err != nil {
					if output != "NIE" {
						log.Printf("cannot convert string '%s' to int; i = %d; error: %v", output, i, err)
					}
					continue
				}
				xor ^= inted
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

func (runner *XorTestRunner) ReadTest(test TestLocation) TestData {
	panic("stub method")
}

func (runner *XorTestRunner) RunTest(test *TestData) TestReport {
	data := TestReport{
		Time:      0,
		Message:   "OK",
		MaxMemory: 0,
		ExitCode:  0,
	}
	cmd := exec.Command(runner.Config.GenericConfig.SolutionPath)
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

func (runner *XorTestRunner) CheckResult(data *TestData, report *TestReport) TestResult {
	panic("stub method")
}
