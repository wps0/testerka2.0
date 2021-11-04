package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
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


var configPath string
var defaultConfig = runners.TestRunnerConfig{
	InputDataDir:           "C:\\Users\\admin\\Downloads\\ezt2\\in",
	OutputDataDir:          "C:\\Users\\admin\\Downloads\\ezt2\\out",
	TestIdRegexp: "([0-9]+)\\.",
	SolutionPath:           "C:\\Users\\admin\\Desktop\\fsfs.exe",
	TimeMeasurementBinPath: "",
}

func InitConfig(configType interface{}) interface{} {
	var err error
	flag.StringVar(&configPath, "cfg", "./config.json", "The path to a config file.")
	flag.Parse()

	cfg, err := loadConfiguration(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration! Error: %s\n", err)
		return runners.TestRunnerConfig{}
	}
	log.Printf("Config loaded!")
	return cfg
}

func loadConfiguration(configPath string) (runners.TestRunnerConfig, error) {
	conf := &runners.TestRunnerConfig{}
	cfgFile, err := os.Open(configPath)
	if err != nil {
		return runners.TestRunnerConfig{}, err
	}
	defer cfgFile.Close()
	d := json.NewDecoder(cfgFile)
	err = d.Decode(conf)
	if err != nil {
		return runners.TestRunnerConfig{}, err
	}
	return *conf, nil
}

func main() {
	log.Printf("Starting testerkav2.0...")
	var config = InitConfig(runners.TestRunnerConfig{}).(runners.TestRunnerConfig)

	regex, err := regexp.Compile(config.TestIdRegexp)
	if err != nil {
		log.Fatalln(err)
	}
	config.TestIdRegexpInternal = *regex
	runner := runners.SimpleTestRunner{}
	runner.Init(config)
	log.Printf("Waiting for the tests to finish...")
	runner.WaitGroup.Wait()
	log.Printf("Finished!")
}
