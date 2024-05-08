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

type JsonData struct {
	StartPos string   `json:"startpos"`
	Moves    []string `json:"moves"`
}

func main() {
	var configPath string
	var dirtestPath string
	var testFile string

	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath = homeDir + "/.config/perfttester/config.yml"
	} else {
		configPath = "perftester.yml"
	}

	flag.StringVar(&configPath, "c", configPath, "path to configuration file")
	flag.StringVar(&dirtestPath, "d", "perfttests", "tests file to run")
	flag.StringVar(&testFile, "t", "", "tests file to run")
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
		dir, err := os.Open(dirtestPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		files, err := dir.Readdir(0)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		globalSuccess := true
		for _, file := range files {
			testFile = dirtestPath + string(os.PathSeparator) + file.Name()
			err, success := runTest(exePath, testFile, *verbose)
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
		err, success := runTest(exePath, testFile, *verbose)
		if err != nil {
			fmt.Println(err)
		}
		if !success {
			os.Exit(1)
		}
	}
}

func runTest(engine string, testfile string, verbose bool) (error, bool) {
	// retrieve testData
	f, err := os.Open(testfile)
	if err != nil {
		return err, false
	}
	defer f.Close()
	var testData JsonData
	if err := json.NewDecoder(f).Decode(&testData); err != nil {
		return err, false
	}

	// run engine
	cmd := exec.Command(engine, "perfttest", testData.StartPos)
	var cmdOut, cmdErr bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	if err := cmd.Run(); err != nil {
		return err, false
	}

	// parse testResult
	var testResult JsonData
	if err := json.Unmarshal(cmdOut.Bytes(), &testResult); err != nil {
		return err, false
	}

	return nil, checkResults(filepath.Base(testfile), testData, testResult, verbose)
}

func checkResults(label string, expected JsonData, got JsonData, verbose bool) bool {
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
