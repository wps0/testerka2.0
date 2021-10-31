package runners

import "regexp"

type TestRunnerConfiguration struct {
	InputDataDir string
	OutputDataDir string
	// This regex applied to the filename uniquely identifies a test
	TestIdRegexp regexp.Regexp
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
	Init(cfg TestRunnerConfiguration)

	TestReader(store *TestStore)
	TestDealer(store *TestStore)

	ReadTest() TestData
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
