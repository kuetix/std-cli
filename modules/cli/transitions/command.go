package transitions

import (
	"bytes"
	"flag"
	"fmt"

	"github.com/kuetix/engine/pkg/domain"
	"github.com/kuetix/engine/pkg/domain/interfaces"
	"github.com/kuetix/engine/pkg/workflow"
	. "github.com/kuetix/std-cli/internal"
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
	cfg := config["config"].(map[string]interface{})
	helpText := cfg["usage"].(string) + "\n"
	options := GetFlags(flags)
	if options["help"].(bool) {
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
