package transitions

import (
	"bytes"
	"flag"
	"fmt"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kuetix/engine"
	"github.com/kuetix/engine/engine/domain"
	"github.com/kuetix/engine/engine/domain/interfaces"
	"github.com/kuetix/engine/engine/workflow"
	"github.com/kuetix/logger"
	. "github.com/kuetix/std-cli/modules/cli/helpers"
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

//goland:noinspection GoUnusedParameter
func (c *cliTransitions) InitCommand(command string, flags map[string]interface{}) (r domain.FlowStepResult) {
	context := c.Ctx.WorkflowContext
	parent := context.Value("parent").(*cliTransitions)
	var ok bool
	if len(flags) > 0 {
		opts := GetFlags(flags)
		if len(opts) > 0 {
			var v string
			if v, ok = opts["modules"].(string); ok {
				parent.modulesPath = v
			}
			if v, ok = opts["workflows"].(string); ok {
				parent.workflowsPath = v
			}
		}
	}

	r.Success = true
	return
}

func (c *cliTransitions) RegisterCommands(
	modulesPath string,
	workflowsPath string,
	version string,
	buildTime string,
	defaultCommand string,
	groups map[string]interface{},
) (r domain.FlowStepResult) {
	c.modulesPath = modulesPath
	c.workflowsPath = workflowsPath
	c.version = version
	c.buildTime = buildTime

	app := c.Ctx.Engine.GetApplication()

	requestedCommand, args, options := GetArgs()
	if requestedCommand == "" {
		requestedCommand = defaultCommand
	}
	c.fs = make(map[string]*flag.FlagSet)
	c.commands = make(map[string]interface{})
	var each []interface{}
	global := make(map[string]interface{})
	eachNames := make(map[string]string)
	if gs, ok := groups["*"].([]interface{}); ok {
		for _, g := range gs {
			if e, ok := g.(map[string]interface{}); ok {
				if l, ok := e["*"].(map[string]interface{})["options"].([]interface{}); ok {
					each = append(each, l...)
				}
				if workflowInit, ok := e["*"].(map[string]interface{})["init"].(string); ok {
					initFs := flag.NewFlagSet(workflowInit, flag.ContinueOnError)
					flags := c.getOptions(each, initFs)
					app.Env.Options.Context["parent"] = c
					commandBase := filepath.Base(workflowInit)
					app.Env.Options.Context["command"] = commandBase
					app.Env.Options.Context["flags"] = flags
					largs := slices.Clone(args)
					appOpts := c.Ctx.Engine.GetApplication().Env.Options
					config := map[string]interface{}{
						"flags": flags,
					}
					_, responses := c.runWorkflow(commandBase, workflowInit, appOpts, config, largs, false, false, false)
					for _, response := range responses {
						if response.Error != nil {
							logger.Errorf("Error initializing command: %s", response.Error.Error())
						}
					}
				}
			}
		}

		if len(each) > 0 {
			for _, v := range each {
				if o, ok := v.(map[string]interface{}); ok {
					// {"long": "modules", "short": "mp", "value": "modules", "usage": "Path to modules directory"},
					eachNames[o["long"].(string)] = o["short"].(string)
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
						command = strings.TrimSpace(command)
						subCommand = strings.TrimSpace(subCommand)
						if subCommand != "" {
							cmd = command + "." + subCommand
						} else {
							cmd = command
						}
						c.fs[cmd] = flag.NewFlagSet(command, flag.ContinueOnError)
						c.fs[cmd].SetOutput(&c.buf)
						commandConfig = c.resolveCommandConfig(commandConfig.(map[string]interface{}), commandsMap)
						c.commands[cmd] = map[string]interface{}{
							"workflow": commandConfig.(map[string]interface{})["workflow"],
							"config":   commandConfig,
							"flags":    flags,
							"flagSet":  c.fs[cmd],
						}
						if requestedCommand != defaultCommand {
							if requestedCommand != cmd {
								continue
							}
						}
						if optionsSlice, ok := commandConfig.(map[string]interface{})["options"].([]interface{}); ok {
							if len(each) > 0 {
								for _, v := range each {
									optionsSlice = append(optionsSlice, v)
								}
							}
							flags = c.getOptions(optionsSlice, c.fs[cmd])
							c.commands[cmd] = map[string]interface{}{
								"workflow": commandConfig.(map[string]interface{})["workflow"],
								"config":   commandConfig,
								"flags":    flags,
								"flagSet":  c.fs[requestedCommand],
							}
							_ = c.fs[cmd].Parse(options)
							for l := range eachNames {
								if _, ok := flags[l]; !ok {
									continue
								}
								option := flags[l]
								global[l] = GetFlag(option)
							}
						}
					}
				}
			}
		}
	}

	app.Env.Options.Context["commands"] = c.commands
	parts := strings.Split(requestedCommand, ".")
	app.Env.Options.Context["requestedCommand"] = map[string]interface{}{
		"command": requestedCommand,
		"parts":   parts,
		"args":    args,
		"options": options,
		"global":  global,
	}

	requestedCommands := []string{
		requestedCommand,
	}
	cmd := parts[0]
	for i := 1; i < len(parts); i++ {
		repeat := strings.Repeat(".*", i)
		requestedCommands = append(requestedCommands, strings.Join([]string{cmd, repeat}, ""))
	}

	var ok bool
	for _, v := range requestedCommands {
		_, ok = c.fs[v]
		if ok {
			requestedCommand = v
			ok = true
			break
		}
	}

	if !ok {
		r.Success = false
		r.Error = fmt.Errorf("Command %s not found", requestedCommand)
		r.Response = map[string]interface{}{
			"commands":         c.commands,
			"workflow":         "",
			"requestedCommand": requestedCommand,
			"config":           nil,
			"args":             args,
			"options":          options,
			"global":           global,
		}
		return
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
		"global":   global,
	}

	return
}

func (c *cliTransitions) getOptions(optionsSlice []interface{}, fs *flag.FlagSet) (flags map[string]interface{}) {
	flags = make(map[string]interface{})
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
				flags[long] = BoolArg(valueBool, FlagBool(fs, long, short, usage, valueBool)...)
			case "int":
				flags[long] = IntArg(valueInt, FlagInt(fs, long, short, usage, valueInt)...)
			case "string":
				flags[long] = StringArg(value, FlagString(fs, long, short, usage, value)...)
			default:
				flags[long] = StringArg(value, FlagString(fs, long, short, usage, value)...)
			}
		}
	}

	return flags
}

func (c *cliTransitions) UnregisteredCommand(value any) (r domain.FlowStepResult) {
	r.Success = true

	if v, ok := value.(*workflow.WorkerResponse); ok {
		if res, ok := v.Response.(map[string]interface{}); ok {
			if cmd, ok := res["requestedCommand"].(string); ok {
				r.Response = map[string]interface{}{
					"message":          fmt.Sprintf("Command '%s' not found. Use 'help' command to see available commands.", cmd),
					"requestedCommand": cmd,
					"availableCommands": func() []string {
						cmds := make([]string, 0, len(c.commands))
						for cmd := range c.commands {
							cmds = append(cmds, cmd)
						}
						return cmds
					}(),
				}
				return
			}
		}
	}

	if v, ok := value.(map[string]interface{}); ok {
		if res, ok := v["Result"].(map[string]interface{}); ok {
			if cmd, ok := res["requestedCommand"].(string); ok {
				r.Response = map[string]interface{}{
					"message":          fmt.Sprintf("Command '%s' not found. Use 'help' command to see available commands.", cmd),
					"requestedCommand": cmd,
					"availableCommands": func() []string {
						cmds := make([]string, 0, len(c.commands))
						for cmd := range c.commands {
							cmds = append(cmds, cmd)
						}
						return cmds
					}(),
				}
				return
			}
		}
	}

	return
}

// WorkflowExecutor executes a WSL workflow for an HTTP config
func (c *cliTransitions) WorkflowExecutor(command, workflowPath string, config map[string]interface{}, args []string, verbose bool, debug bool, quiet bool) (result domain.FlowStepResult) {
	options := c.Ctx.Engine.GetApplication().Env.Options
	workflowPath, responses := c.runWorkflow(command, workflowPath, options, config, args, verbose, debug, quiet)

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
				if strings.Contains(s, " trace: ") && debug != true {
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

func (c *cliTransitions) runWorkflow(command string, workflowPath string, options *domain.Options, config map[string]interface{}, args []string, verbose bool, debug bool, quiet bool) (string, map[string]*workflow.WorkerResponse) {
	// Parse request into workflow arguments
	context := maps.Clone(options.Context)
	settings := maps.Clone(options.Settings)
	options = &domain.Options{
		EngineName:      options.EngineName,
		ConfigName:      options.ConfigName,
		Verbose:         options.Verbose,
		Quiet:           options.Quiet,
		Amount:          options.Amount,
		Retry:           options.Retry,
		RetryDelay:      options.RetryDelay,
		RestartPolicy:   options.RestartPolicy,
		Workflow:        workflowPath,
		Version:         options.Version,
		BuildTime:       options.BuildTime,
		LogPath:         options.LogPath,
		Config:          options.Config,
		Args:            options.Args,
		Context:         context,
		Settings:        settings,
		EmbedFS:         options.EmbedFS,
		EmbedFSRootPath: options.EmbedFSRootPath,
	}
	context["Workflow"] = workflowPath
	context["args"] = args
	context["config"] = config
	context["command"] = command
	context["flags"] = config["flags"]

	if verbose {
		logger.EnableInfo()
	}

	if debug {
		logger.EnableDebug()
	}

	// Execute the workflow
	workflowPath = filepath.Join(c.workflowsPath, workflowPath)
	args = []string{
		// Add configuration to args
		fmt.Sprintf("command=%s", command),
		fmt.Sprintf("modulesPath=%s", c.modulesPath),
		fmt.Sprintf("workflowsPath=%s", c.workflowsPath),
		fmt.Sprintf("version=%s", c.version),
		fmt.Sprintf("buildTime=%s", c.buildTime),
	}
	responses := engine.RunWorkflow("production", &domain.Options{
		EngineName:      "kuetix-cli",
		ConfigName:      "cli",
		Verbose:         verbose || debug,
		Quiet:           quiet,
		Amount:          1,
		Retry:           1,
		RetryDelay:      0,
		RestartPolicy:   options.RestartPolicy,
		Workflow:        workflowPath,
		Version:         options.Version,
		BuildTime:       options.BuildTime,
		LogPath:         options.LogPath,
		Config:          options.Config,
		Args:            args,
		Settings:        options.Settings,
		Context:         context,
		EmbedFS:         options.EmbedFS,
		EmbedFSRootPath: options.EmbedFSRootPath,
	})

	return workflowPath, responses
}

func (c *cliTransitions) resolveCommandConfig(commandConfig, commandsMap map[string]interface{}) map[string]interface{} {
	if extends, ok := commandConfig["$extends"]; ok {
		if parent, ok := commandsMap[extends.(string)]; ok {
			parentMap := parent.(map[string]interface{})
			mergedConfig := make(map[string]interface{})
			for k, v := range parentMap {
				if k == "$extends" {
					v = c.resolveCommandConfig(parentMap, commandsMap)
					if v == nil {
						v = parentMap
					} else {
						for vk, vv := range v.(map[string]interface{}) {
							mergedConfig[vk] = vv
						}
					}
					continue
				}
				mergedConfig[k] = v
			}
			for k, v := range commandConfig {
				if k == "$extends" {
					continue
				}
				mergedConfig[k] = v
			}
			return mergedConfig
		}
	}

	return commandConfig
}
