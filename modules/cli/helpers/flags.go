package helpers

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kuetix/engine/engine/domain"
)

func FlagInt(fs *flag.FlagSet, long, short, usage string, value int) []*int {
	l := fs.Int(long, value, usage)
	s := fs.Int(short, value, usage+" (shorthand)")
	return []*int{l, s}
}

func FlagBool(fs *flag.FlagSet, long, short, usage string, value bool) []*bool {
	l := fs.Bool(long, value, usage)
	s := fs.Bool(short, value, usage+" (shorthand)")
	return []*bool{l, s}
}

func FlagString(fs *flag.FlagSet, long, short, usage, value string) []*string {
	l := fs.String(long, value, usage)
	s := fs.String(short, value, usage+" (shorthand)")
	return []*string{l, s}
}

func IntArg(defaultValue int, flags ...*int) func() *int {
	return func() *int {
		for _, f := range flags {
			if f != nil && *f != defaultValue {
				return f
			}
		}
		return &defaultValue
	}
}

func BoolArg(defaultValue bool, flags ...*bool) func() *bool {
	return func() *bool {
		for _, f := range flags {
			if *f != defaultValue {
				return f
			}

		}
		return &defaultValue
	}
}

func StringArg(defaultValue string, flags ...*string) func() *string {
	return func() *string {
		for _, f := range flags {
			if strings.TrimSpace(*f) != "" && strings.TrimSpace(*f) != defaultValue {
				*f = strings.TrimSpace(*f)
				return f
			}
		}
		return &defaultValue
	}
}

func GetArgs() (mainCommand string, command string, args []string, options []string) {
	args = []string{}
	options = []string{}
	nextIsOptionValue := false
	foundCommand := false
	foundSubcommand := false
	mainCommand = ""

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		if strings.HasPrefix(arg, "-") {
			nextIsOptionValue = false
		}

		if nextIsOptionValue {
			nextIsOptionValue = false
			options = append(options, arg)
			continue
		}

		// Found the command (e.g., "add")
		if !foundCommand && !strings.HasPrefix(arg, "-") {
			foundCommand = true
			command = arg
			mainCommand = arg
			continue
		}

		// Found the subcommand (e.g., "module")
		if foundCommand && !foundSubcommand && !strings.HasPrefix(arg, "-") {
			foundSubcommand = true
			command = command + "." + arg
			continue
		}

		if foundSubcommand && !strings.HasPrefix(arg, "-") {
			args = append(args, arg)
		} else {
			nextIsOptionValue = true
			options = append(options, arg)
		}
	}

	return mainCommand, command, args, options
}

func GetFlag(option any) (value any) {
	switch option.(type) {
	case func() *string:
		value = option.(func() *string)()
		if value.(*string) == nil {
			value = ""
		} else {
			value = *value.(*string)
		}
	case func() *int:
		value = option.(func() *int)()
		if value.(*int) == nil {
			value = 0
		} else {
			value = *value.(*int)
		}
	case func() *bool:
		value = option.(func() *bool)()
		if value.(*bool) != nil && *value.(*bool) == true {
			value = true
		} else {
			value = false
		}
	default:
		value = nil
	}

	return
}

func GetFlags(flags map[string]interface{}) (r map[string]interface{}) {
	r = make(map[string]interface{})
	for name, option := range flags {
		r[name] = GetFlag(option)
	}

	return
}

func GetUsage(app domain.Application, usage string, flagSet *flag.FlagSet, rootPath string) string {
	var helpText string = ""
	if usage != "" {
		if strings.HasPrefix(usage, "file://") {
			usage = strings.Trim(usage, "\n\n\t ")
			file := strings.TrimPrefix(usage, "file://")
			filePath := filepath.Join(rootPath, file)
			content := ReadFileAsString(app, filePath)
			if !strings.HasPrefix(content, "err:") {
				usage = content
			}
		}

		if flagSet != nil {
			var buf bytes.Buffer
			flagSet.SetOutput(&buf)
			flagSet.Usage()
			helpText = buf.String() + "\n"
		}

		helpText = usage + "\n"

		return helpText
	}

	return helpText
}

//goland:noinspection GoUnusedExportedFunction
func ReadFileAsString(app domain.Application, path string) string {
	var candidates = []string{
		fmt.Sprintf("%s/%s", app.EmbedFSRootPath, path),
		fmt.Sprintf("%s", path),
	}
	for _, candidate := range candidates {
		content, err := app.EmbedFS.ReadFile(candidate)
		if err == nil {
			return string(content)
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "err:" + err.Error()
	}
	return string(content)
}
