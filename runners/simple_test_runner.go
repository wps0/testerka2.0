package runners

type SimpleTestRunner struct{}

func (SimpleTestRunner) Init(cfg TestRunnerConfiguration) {
	panic("implement me")
}

func (SimpleTestRunner) TestReader() {
	panic("implement me")
}

func (SimpleTestRunner) TestDealer() {
	panic("implement me")
}

func (SimpleTestRunner) ReadTest() TestConfiguration {
	panic("implement me")
}

func (SimpleTestRunner) RunTest(test TestConfiguration) {
	panic("implement me")
}

func (SimpleTestRunner) CheckResult(report TestReport) TestResult {
	panic("implement me")
}

