package runners

type SimpleTestRunner struct {

}

func (runner *SimpleTestRunner) Init(cfg TestRunnerConfiguration) {
	panic("implement me")
}

func (runner *SimpleTestRunner) TestReader(store *TestStore) {
	panic("implement me")
}

func (runner *SimpleTestRunner) TestDealer(store *TestStore) {
	panic("implement me")
}

func (runner *SimpleTestRunner) ReadTest() TestData {
	panic("implement me")
}

func (runner *SimpleTestRunner) RunTest(test *TestData) {
	panic("implement me")
}

func (runner *SimpleTestRunner) CheckResult(data *TestData, report *TestReport) TestResult {
	panic("implement me")
}

