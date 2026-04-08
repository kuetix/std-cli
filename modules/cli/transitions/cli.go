package transitions

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kuetix/engine"
	"github.com/kuetix/engine/boot"
	"github.com/kuetix/engine/pkg/domain"
	"github.com/kuetix/engine/pkg/domain/interfaces"
	"github.com/kuetix/engine/pkg/workflow"
	. "github.com/kuetix/std-cli/internal"
)

type cliTransitions struct {
	workflow.BaseServiceTransition
	modulesPath   string
	workflowsPath string
	version       string
	buildTime     string
	buf           bytes.Buffer
	fs            map[string]*flag.FlagSet
	commands      map[string]interface{}
}

func NewCliTransition() interfaces.ServiceTransitions { return &cliTransitions{} }

func (c *cliTransitions) RegisterCommands(
	modulesPath string,
	workflowsPath string,
	version string,
	buildTime string,
	groups map[string]interface{},
) (r domain.FlowStepResult) {
	c.modulesPath = modulesPath
	c.workflowsPath = workflowsPath
	c.version = version
	c.buildTime = buildTime
	// c.output = io.Writer()

	requestedCommand, args, options := GetArgs()
	if requestedCommand == "" {
		requestedCommand = "help"
	}
	c.fs = make(map[string]*flag.FlagSet)
	c.commands = make(map[string]interface{})
	var each []interface{}
	if gs, ok := groups["*"].([]interface{}); ok {
		for _, g := range gs {
			if e, ok := g.(map[string]interface{}); ok {
				if l, ok := e["*"].(map[string]interface{})["options"].([]interface{}); ok {
					each = l
				}
			}
		}
	}
	delete(groups, "*")
	for command, commandsList := range groups {
		if commandsConfigs, ok := commandsList.([]interface{}); ok {
			for _, commands := range commandsConfigs {
				if commandsMap, ok := commands.(map[string]interface{}); ok {
					flags := make(map[string]interface{})
					for subCommand, commandConfig := range commandsMap {
						var cmd string
						if optionsSlice, ok := commandConfig.(map[string]interface{})["options"].([]interface{}); ok {
							if len(each) > 0 {
								for _, v := range each {
									optionsSlice = append(optionsSlice, v)
								}
							}
							command = strings.TrimSpace(command)
							subCommand = strings.TrimSpace(subCommand)
							if subCommand != "" {
								cmd = command + "." + subCommand
							} else {
								cmd = command
							}
							if requestedCommand != "help" {
								if requestedCommand != cmd {
									continue
								}
							}
							c.fs[cmd] = flag.NewFlagSet(command, flag.ContinueOnError)
							c.fs[cmd].SetOutput(&c.buf)
							for _, opts := range optionsSlice {
								if o, ok := opts.(map[string]interface{}); ok {
									// {"long": "modules", "short": "mp", "value": "modules", "usage": "Path to modules directory"},
									long, ok := o["long"].(string)
									if !ok {
										long = ""
									}
									short, ok := o["short"].(string)
									if !ok {
										short = ""
									}
									value, ok := o["value"].(string)
									if !ok {
										value = ""
									}
									valueBool, ok := o["value"].(bool)
									if !ok {
										valueBool = false
									}
									valueInt, ok := o["value"].(int)
									if !ok {
										valueInt = 0
										// Try to convert float to int if it's a float
									}
									if valueFloat, ok := o["value"].(float64); ok {
										valueInt = int(valueFloat)
									}
									usage, ok := o["usage"].(string)
									if !ok {
										usage = ""
									}
									optionType, ok := o["type"].(string)
									if !ok {
										optionType = "string"
									}
									switch optionType {
									case "bool":
										flags[long] = BoolArg(valueBool, FlagBool(c.fs[cmd], long, short, usage, valueBool)...)
									case "int":
										flags[long] = IntArg(valueInt, FlagInt(c.fs[cmd], long, short, usage, valueInt)...)
									case "string":
										flags[long] = StringArg(value, FlagString(c.fs[cmd], long, short, usage, value)...)
									default:
										flags[long] = StringArg(value, FlagString(c.fs[cmd], long, short, usage, value)...)
									}
								}
								c.commands[cmd] = map[string]interface{}{
									"workflow": commandConfig.(map[string]interface{})["workflow"],
									"config":   commandConfig,
									"flags":    flags,
									"flagSet":  c.fs[requestedCommand],
								}
							}
							err := c.fs[cmd].Parse(options)
							if err != nil {
								c.buf.Reset()
								c.fs[cmd].Usage()
								fmt.Printf("%s\nError: %s\n", c.buf.String(), err)
								os.Exit(1)
								return
							}
						}
					}
				}
			}
		}
	}

	r.Success = true
	c.commands[requestedCommand].(map[string]interface{})["flagSet"] = c.fs[requestedCommand]
	commandWorkflow := c.commands[requestedCommand].(map[string]interface{})["workflow"]
	r.Response = map[string]interface{}{
		"commands": c.commands,
		"workflow": commandWorkflow,
		"command":  requestedCommand,
		"config":   c.commands[requestedCommand],
		"args":     args,
		"options":  options,
	}

	return
}

// WorkflowExecutor executes a WSL workflow for an HTTP config
func (c *cliTransitions) WorkflowExecutor(command, workflowPath string, config map[string]interface{}, args []string) (result domain.FlowStepResult) {
	options := c.Ctx.Engine.GetApplication().Env.Options

	// Parse request into workflow arguments
	options.Args = []string{
		// Add configuration to args
		fmt.Sprintf("command=%s", command),
		fmt.Sprintf("modulesPath=%s", c.modulesPath),
		fmt.Sprintf("workflowsPath=%s", c.workflowsPath),
		fmt.Sprintf("version=%s", c.version),
		fmt.Sprintf("buildTime=%s", c.buildTime),
	}

	context := map[string]interface{}{
		"command": command,
		"config":  config,
		"flags":   config["flags"],
		"args":    args,
	}

	// Execute the workflow
	workflowPath = filepath.Join(c.workflowsPath, workflowPath)
	responses := engine.RunWorkflow(&boot.Options{
		EngineName:    "kapi-api",
		ConfigName:    "http",
		Verbose:       options.Verbose,
		Quiet:         options.Quiet,
		Amount:        1,
		Retry:         1,
		RetryDelay:    0,
		RestartPolicy: options.RestartPolicy,
		Workflow:      workflowPath,
		Version:       options.Version,
		BuildTime:     options.BuildTime,
		LogPath:       options.LogPath,
		Config:        options.Config,
		Args:          options.Args,
		Settings:      options.Settings,
		Context:       context,
	})

	var response *workflow.WorkerResponse
	responseRef, ok := responses[workflowPath]
	if ok {
		response = responseRef
	}
	base := filepath.Base(workflowPath)
	responseRef, ok = responses[base]
	if ok {
		response = responseRef
	}
	if response == nil {
		ok = false
		for _, resp := range responses {
			responseRef = resp
			ok = true
			break
		}
	}
	if ok {
		response = responseRef
	}

	// Check if workflow execution was successful
	if response == nil || !response.IsSuccess() {
		result.StatusCode = http.StatusInternalServerError
		var errorMessages []string = make([]string, 0)
		if response != nil && response.Error != nil {
			result.StatusCode = response.StatusCode
			issues := response.Error.Errors()
			for _, issue := range issues {
				s := issue.Error()
				if strings.Contains(s, " trace: ") && c.Ctx.Engine.GetApplication().Env.Config.Application.Debug != true {
					continue
				}
				errorMessages = append(errorMessages, s)
			}
			result.Error = response.Error
		}
		if len(errorMessages) == 0 {
			errorMessages = append(errorMessages, "Workflow execution failed with unknown error")
		}
		RespondErrors(errorMessages, result.StatusCode)
		result.Success = false
		return
	}

	// Extract response from workflow result
	if response.Response != nil {
		// Send the workflow response back to client
		RespondSuccess(response.Response)
	} else {
		RespondSuccess(map[string]interface{}{
			"success": true,
			"message": "Workflow executed successfully",
		})
	}

	result.Success = true
	result.Response = response.Response
	return
}
