package main

import (
	"log"
	"regexp"
	"testerka2.0/runners"
)

// test runner - zestaw metod odpalajacych testy; jakis interface

//
//     main ---> tests_runner <---> events_hooks <---> frontend
//                    /\
//                    |
//                   \/
//               test_worker


// wybor test runnera
// odpalenie testow na test runnerze

func main() {
	log.Printf("Starting testerkav2.0...")
	regex, err := regexp.Compile("([0-9]+)\\.")
	if err != nil {
		log.Fatalln(err)
	}
	runner := runners.SimpleTestRunner{}
	config := runners.TestRunnerConfig{
		InputDataDir:           "C:\\Users\\admin\\Downloads\\ezt2\\in",
		OutputDataDir:          "C:\\Users\\admin\\Downloads\\ezt2\\out",
		TestIdRegexp:           *regex,
		SolutionPath:           "C:\\Users\\admin\\Desktop\\fsfs.exe",
		TimeMeasurementBinPath: "",
	}
	runner.Init(config)
	log.Printf("Waiting for the tests to finish...")
	runner.WaitGroup.Wait()
	log.Printf("Finished!")
}
