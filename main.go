// SPDX-FileCopyrightText: 2024 vinymeuh
// SPDX-License-Identifier: MIT
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Engines []ConfigEngine  `yaml:"engines"`
	Tests   []ConfigDirTest `yaml:"dirtests"`
}

type ConfigEngine struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type ConfigDirTest struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

func NewConfig(configPath string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

type TestsDefinition struct {
	StartPos string               `json:"startpos"`
	Moves    []string             `json:"moves"`
	Nodes    []TestNodeDefinition `json:"nodes,omitempty"`
}

type TestNodeDefinition struct {
	Depth int `json:"depth"`
	Nodes int `json:"nodes"`
}

func main() {
	var configPath string
	var dirtestPath string
	var testFile string

	configPath = ".perfttester.yml"

	flag.StringVar(&configPath, "c", configPath, "path to configuration file")
	flag.StringVar(&dirtestPath, "d", "perft", "test directory to use")
	flag.StringVar(&testFile, "t", "", "test file to run")
	verbose := flag.Bool("v", false, "verbose")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-c config] [-d dirtest] [-t testfile] [-v] /path/to/engine\n", os.Args[0])
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
	}
	exePath := os.Args[len(os.Args)-1]

	if config, err := NewConfig(configPath); err == nil {
		if i := slices.IndexFunc(config.Engines, func(e ConfigEngine) bool { return e.Name == exePath }); i >= 0 {
			exePath = config.Engines[i].Path
		}
		if i := slices.IndexFunc(config.Tests, func(t ConfigDirTest) bool { return t.Name == dirtestPath }); i >= 0 {
			dirtestPath = config.Tests[i].Path
		}
	}
	fmt.Fprintln(os.Stdout, "Engine path    :", exePath)
	fmt.Fprintln(os.Stdout, "Tests directory:", dirtestPath)

	// run tests in dir or file mode
	if testFile == "" {
		files, err := os.ReadDir(dirtestPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}

		globalSuccess := true
		for _, file := range files {
			testFile = dirtestPath + string(os.PathSeparator) + file.Name()
			err, success := runTests(exePath, testFile, *verbose)
			if err != nil {
				fmt.Println(err)
			}
			if !success {
				globalSuccess = false
			}
		}
		if !globalSuccess {
			os.Exit(1)
		}
	} else {
		testFile = dirtestPath + string(os.PathSeparator) + testFile
		err, success := runTests(exePath, testFile, *verbose)
		if err != nil {
			fmt.Println(err)
		}
		if !success {
			os.Exit(1)
		}
	}
}

func runTests(engine string, testfile string, verbose bool) (error, bool) {
	// retrieve testData
	f, err := os.Open(testfile)
	if err != nil {
		return err, false
	}
	defer f.Close()
	var testsDefinition TestsDefinition
	if err := json.NewDecoder(f).Decode(&testsDefinition); err != nil {
		return err, false
	}

	// depth = 1
	result, err := runTestDepth1(engine, testsDefinition.StartPos)
	if err == nil {
		if ok := checkResultsDepth1(filepath.Base(testfile), testsDefinition, result, verbose); ok == false {
			return nil, false
		}
	} else {
		return err, false
	}

	// depth > 1
	for _, testNode := range testsDefinition.Nodes {
		result, err := runTestDepthN(engine, testsDefinition.StartPos, testNode.Depth)
		if err == nil {
			if ok := checkResultsDepthN(filepath.Base(testfile), testNode, result); ok == false {
				return nil, false
			}
		} else {
			return err, false
		}
	}

	return nil, true
}

type TestResultDepth1 struct {
	StartPos string   `json:"startpos"`
	Moves    []string `json:"moves"`
}

func runTestDepth1(engine string, startpos string) (TestResultDepth1, error) {
	cmd := exec.Command(engine, "perfttest", startpos, "1")
	var cmdOut, cmdErr bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	if err := cmd.Run(); err != nil {
		return TestResultDepth1{}, err
	}
	var testResult TestResultDepth1
	if err := json.Unmarshal(cmdOut.Bytes(), &testResult); err != nil {
		fmt.Println(testResult)
		return TestResultDepth1{}, err
	}
	return testResult, nil
}

func checkResultsDepth1(label string, expected TestsDefinition, got TestResultDepth1, verbose bool) bool {
	success := true
	var verboseErrors []string

	// check moves count
	movesCountExpected := len(expected.Moves)
	movesCountGot := len(got.Moves)
	if movesCountGot != movesCountExpected {
		success = false
		verboseErrors = append(verboseErrors,
			fmt.Sprintf("%s -- KO -- moves count mismatch, expected=%d, got=%d", label, movesCountExpected, movesCountGot),
		)
	}

	// check we have all expected moves
	for _, mExpected := range expected.Moves {
		if slices.IndexFunc(got.Moves, func(mGot string) bool { return mGot == mExpected }) < 0 {
			success = false
			verboseErrors = append(verboseErrors,
				fmt.Sprintf("%s -- KO -- missing an expected move %s", label, mExpected),
			)
		}
	}

	// check if we have unexpected moves
	for _, mGot := range got.Moves {
		if slices.IndexFunc(expected.Moves, func(mExpected string) bool { return mExpected == mGot }) < 0 {
			success = false
			verboseErrors = append(verboseErrors,
				fmt.Sprintf("%s -- KO -- got an unexpected move %s", label, mGot),
			)
		}
	}

	// final report
	if success {
		fmt.Fprintf(os.Stdout, "%s -- OK -- position sfen %s\n", label, expected.StartPos)
	} else {
		fmt.Fprintf(os.Stdout, "%s -- KO -- position sfen %s\n", label, expected.StartPos)
		if verbose {
			for _, msg := range verboseErrors {
				fmt.Fprintln(os.Stdout, msg)
			}
		}
	}
	return success
}

type TestResultDepthN struct {
	Depth int `json:"depth"`
	Nodes int `json:"nodes"`
}

func runTestDepthN(engine string, startpos string, depth int) (TestResultDepthN, error) {
	cmd := exec.Command(engine, "perfttest", startpos, strconv.Itoa(depth))
	var cmdOut, cmdErr bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	if err := cmd.Run(); err != nil {
		fmt.Println(cmd) // FIXME
		return TestResultDepthN{}, err
	}
	var testResult TestResultDepthN
	if err := json.Unmarshal(cmdOut.Bytes(), &testResult); err != nil {
		return TestResultDepthN{}, err
	}
	return testResult, nil
}

func checkResultsDepthN(label string, expected TestNodeDefinition, got TestResultDepthN) bool {
	if got.Nodes != expected.Nodes {
		fmt.Fprintf(os.Stdout, "%s -- KO -- nodes count mismatch at depth %d, expected=%d, got=%d\n", label, expected.Depth, expected.Nodes, got.Nodes)
		return false
	}

	return true
}
