package transitions

import (
	"bytes"
	"flag"
	"fmt"

	"github.com/kuetix/engine/engine/domain"
	"github.com/kuetix/engine/engine/domain/interfaces"
	"github.com/kuetix/engine/engine/workflow"
	"github.com/kuetix/logger"
	. "github.com/kuetix/std-cli/modules/cli/helpers"
)

type commandTransition struct {
	workflow.BaseServiceTransition
	modulesPath   string
	workflowsPath string
	version       string
	buildTime     string
	fs            map[string]*flag.FlagSet
	commands      map[string]interface{}
}

func NewCommandTransition() interfaces.ServiceTransitions { return &commandTransition{} }

//goland:noinspection GoUnusedParameter
func (h *commandTransition) Test(command string, config map[string]interface{}, flags map[string]interface{}) (r domain.FlowStepResult) {
	var helpText string
	cfg := config["config"].(map[string]interface{})
	options := GetFlags(flags)
	logger.Info("Command options: ", options, " Command: ", command)
	if options["help"].(bool) {
		helpText = cfg["usage"].(string) + "\n"
		var buf bytes.Buffer
		flagSet := config["flagSet"].(*flag.FlagSet)
		flagSet.SetOutput(&buf)
		flagSet.Usage()
		helpText += buf.String()
	} else {
		if echo, ok := options["echo"].(string); ok {
			helpText += fmt.Sprintf("%s\n", echo)
		}
	}
	r.Success = true
	r.Response = helpText
	return
}
