package runners

import "regexp"

type TestRunnerConfig struct {
	InputData  string
	OutputData string
	// This regex applied to the filename uniquely identifies a test
	TestIdRegexpInternal regexp.Regexp `json:"-"`
	TestIdRegexp         string

	SolutionPath           string
	TimeMeasurementBinPath string

	SkipFirstN uint
}

type TestData struct {
	InputData      []byte
	ExpectedOutput []byte
}

type TestLocation struct {
	InputFilePath  string
	OutputFilePath string
}

type TestReport struct {
	// in ms
	Time uint
	// in kBs
	MaxMemory uint
	Output    []byte
	ExitCode  int
	Message   string
}

type TestStats struct {
	WACount int
	OKCount int
}

type TestRunner interface {
	Init(cfg TestRunnerConfig)

	TestReader()
	TestRunner()

	GetStats() TestStats

	ReadTest(test TestLocation) TestData
	RunTest(test *TestData) TestReport
	CheckResult(data *TestData, report *TestReport) TestResult
}

type TestStore interface {
	Get(id string) (TestData, error)
	Insert(id string, data TestData) error
	Remove(id string) error
	Size() uint64
	Exists(id string) bool
}

type TestResult struct {
	Status  bool
	Message string
}
