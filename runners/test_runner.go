package runners

import "regexp"

type TestRunnerConfiguration struct {
	InputDataDir string
	OutputDataDir string
	// This regex applied to the filename uniquely identifies a test
	TestIdRegexp regexp.Regexp
}

type TestConfiguration struct {
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

	TestReader()
	TestDealer()

	ReadTest() TestConfiguration
	RunTest(test TestConfiguration)
	CheckResult(report TestReport) TestResult
}

type TestResult struct {
	Status bool
	Message string
}
