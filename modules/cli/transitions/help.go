package transitions

import (
	"bytes"
	"flag"

	"github.com/kuetix/engine/engine/domain"
	"github.com/kuetix/engine/engine/domain/interfaces"
	"github.com/kuetix/engine/engine/workflow"
)

type helpTransition struct {
	workflow.BaseServiceTransition
	modulesPath   string
	workflowsPath string
	version       string
	buildTime     string
	fs            map[string]*flag.FlagSet
	commands      map[string]interface{}
}

func NewHelpTransition() interfaces.ServiceTransitions { return &helpTransition{} }

//goland:noinspection GoUnusedParameter
func (h *helpTransition) Help(command string, config map[string]interface{}, flags map[string]interface{}) (r domain.FlowStepResult) {
	cfg := config["config"].(map[string]interface{})
	helpText := cfg["usage"].(string) + "\n"
	var buf bytes.Buffer
	flagSet := config["flagSet"].(*flag.FlagSet)
	flagSet.SetOutput(&buf)
	flagSet.Usage()
	helpText += buf.String()

	r.Success = true
	r.Response = helpText
	return
}
