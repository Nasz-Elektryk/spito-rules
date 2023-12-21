package checker

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"sync"
	"gopkg.in/yaml.v3"
)

const CONFIG_FILENAME = "spito-rules.yml"
const LOCK_FILENAME = "spito-lock.yml"

func getRuleSetsDir() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return dir + "/.local/state/spito-rules/rule-sets", nil
}

func initRequiredTmpDirs() error {
	dir, err := getRuleSetsDir()
	if err != nil {
		return err
	}
	err = os.MkdirAll(dir, 0700)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	return err
}

func getDefaultRepoPrefix() string {
	// TODO: implement logic for getting default repo prefix
	return "github.com"
}

type RuleSetLocation struct {
	simpleUrl string
}

type DependencyTreeLayout struct {
	Dependencies []string
}

// e.g. from: https://github.com/Nasz-Elektryk/spito-ruleset.git to Nasz-Elektryk/spito-ruleset
func (r *RuleSetLocation) new(identifier string) {
	// check if simpleUrl is url:
	if !strings.Contains(identifier, ".") {
		r.simpleUrl = getDefaultRepoPrefix() + "/" + identifier
		return
	}

	simpleUrl := identifier
	simpleUrl = strings.ReplaceAll(simpleUrl, "https://", "")
	simpleUrl = strings.ReplaceAll(simpleUrl, "http://", "")
	simpleUrl = strings.ReplaceAll(simpleUrl, "www.", "")
	urlLen := len(simpleUrl)

	if simpleUrl[urlLen-1] == '/' {
		simpleUrl = simpleUrl[:urlLen-1]
	}
	// I still wonder whether it is good idea:
	if simpleUrl[urlLen-4:] == ".git" {
		simpleUrl = simpleUrl[:urlLen-4]
	}

	r.simpleUrl = simpleUrl
}

func (r *RuleSetLocation) createDir() error {
	println(r.getRuleSetPath())
	err := os.MkdirAll(r.getRuleSetPath(), 0700)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	return err
}

func (r *RuleSetLocation) getFullUrl() string {
	return "https://" + r.simpleUrl
}

func (r *RuleSetLocation) getRuleSetPath() string {
	dir, err := getRuleSetsDir()
	if err != nil {
		return ""
	}
	return dir + "/" + r.simpleUrl
}

func (r *RuleSetLocation) isRuleSetDownloaded() bool {
	_, err := os.ReadDir(r.getRuleSetPath())
	return !errors.Is(err, fs.ErrNotExist)
}

func (rulesetLocation *RuleSetLocation) createLockfile(rulesInProgress map[string]bool) ([]string, error) {
	configPath := rulesetLocation.getRuleSetPath() + "/" + CONFIG_FILENAME
	configFileContents, error := os.ReadFile(configPath)
	if error != nil {
		return []string{}, error
	}

	var basicDependencyTree DependencyTreeLayout

	error = yaml.Unmarshal(configFileContents, &basicDependencyTree)
	if error != nil {
		return []string{}, error
	}

	var outputDependencyTree DependencyTreeLayout
	outputDependencyTree.Dependencies = make([]string, len(basicDependencyTree.Dependencies))
	copy(outputDependencyTree.Dependencies, basicDependencyTree.Dependencies)

	firstDependencyLocation := RuleSetLocation{}
	
	if len(basicDependencyTree.Dependencies) > 0 {
		firstDependencyLocation.new(strings.Split(basicDependencyTree.Dependencies[0], "@")[0])
	}

	var waitGroup sync.WaitGroup

	for _, dependencyName := range basicDependencyTree.Dependencies {
		waitGroup.Add(1)
		go func(dependencyNameParameter string) {
			defer waitGroup.Done()
			dependencyLocation := RuleSetLocation{}
			dependencyLocation.new(strings.Split(dependencyNameParameter, "@")[0])
			if _, exists := rulesInProgress[dependencyNameParameter]; !exists && !dependencyLocation.isRuleSetDownloaded() {
				rulesInProgress[dependencyNameParameter] = true
				FetchRuleSet(&dependencyLocation)
			}
		}(dependencyName)
	}

	waitGroup.Wait()
	if firstDependencyLocation.simpleUrl != "" {
		toBeAppended, error := firstDependencyLocation.createLockfile(rulesInProgress)
		if error != nil {
			return nil, error
		}
		outputDependencyTree.Dependencies = append(outputDependencyTree.Dependencies, toBeAppended...)
	}

	lockfilePath := rulesetLocation.getRuleSetPath() + "/" + LOCK_FILENAME
	lockfile, error := os.Create(lockfilePath)
	
	if error != nil {
		return []string{}, error
	}
	defer lockfile.Close()

	yamlOutput, error := yaml.Marshal(outputDependencyTree)
	if error != nil {
		return []string{}, error
	}

	lockfile.Write(yamlOutput)

	return outputDependencyTree.Dependencies, nil
}