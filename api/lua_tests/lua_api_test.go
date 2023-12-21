package api_tests

import (
	"fmt"
	"github.com/nasz-elektryk/spito/checker"
	"github.com/nasz-elektryk/spito/cmd/cmdApi"
	"github.com/nasz-elektryk/spito/shared"
	"github.com/nasz-elektryk/spito/vrct"
	"os"
	"testing"
)

func TestLuaApi(t *testing.T) {
	scripts := []string{
		"sysinfo_test.lua",
		"fs_test.lua",
		"rule_require_test.lua",
		"daemon_test.lua",
		"package_test.lua",
	}

	for _, scriptName := range scripts {
		file, err := os.ReadFile(scriptName)
		if err != nil {
			t.Fatal(err)
		}

		ruleVRCT, err := vrct.NewRuleVRCT()
		if err != nil {
			t.Fatal("Failed to initialized rule VRCT", err)
		}

		runtimeData := shared.ImportLoopData{
			VRCT:         *ruleVRCT,
			RulesHistory: shared.RulesHistory{},
			ErrChan:      make(chan error),
			InfoApi:      cmdApi.InfoApi{},
		}

		doesRulePass, err := checker.CheckRuleScript(&runtimeData, string(file))
		if err != nil {
			t.Fatalf("Error occurred: %s", fmt.Sprint(err))
		}

		if !doesRulePass {
			t.Fatalf("Rule %s did not pass!", scriptName)
		}
	}
}