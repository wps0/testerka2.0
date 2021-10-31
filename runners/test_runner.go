package runners

import "regexp"

type TestRunnerConfig struct {
	InputDataDir string
	OutputDataDir string
	// This regex applied to the filename uniquely identifies a test
	TestIdRegexp regexp.Regexp

	SolutionPath string
	TimeMeasurementBinPath string
}

type TestData struct {
	InputData []byte
	ExpectedOutput []byte
}

type TestReport struct {
	// in ms
	Time uint
	// in kBs
	MaxMemory uint
	Output []byte
	ExitCode int
}

type TestRunner interface {
	Init(cfg TestRunnerConfig)

	TestReader()
	TestDealer()

	ReadTest(inputPath string, outputPath string)
	RunTest(test *TestData)
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
	Status bool
	Message string
}
