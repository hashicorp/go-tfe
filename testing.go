package tfe

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

func envLookupInteger(name string) (int, bool) {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return 0, false
	}

	result, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return result, true
}

type testSuiteCI struct {
	mu        sync.Mutex
	testNames map[string]int
}

func (s *testSuiteCI) listTestsCI() (map[string]int, error) {
	cmd := exec.Command("go", "test", "./...", "-list=.", "-tags=integration")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list all test files. Are you using go1.19+?: %w", err)
	}

	result := make(map[string]int)
	index := 0
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "Test") {
			result[line] = index
			index += 1
		}
	}
	return result, nil
}

func (s *testSuiteCI) init() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.testNames == nil {
		var err error
		s.testNames, err = s.listTestsCI()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *testSuiteCI) InCurrentNode(name string) (bool, error) {
	if nodeIndex, ok := envLookupInteger("CI_NODE_INDEX"); ok {
		if nodeTotal, ok := envLookupInteger("CI_NODE_TOTAL"); ok {
			err := s.init()
			if err != nil {
				return false, err
			}

			testIndex, ok := s.testNames[name]
			if !ok {
				return false, fmt.Errorf("%s was not found in the list of tests", name)
			}

			if testIndex%nodeTotal != nodeIndex {
				return false, nil
			}
		}
	}

	return true, nil
}
