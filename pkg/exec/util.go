package exec

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mickael-carl/sophons/pkg/variables"
	"github.com/nikolalohinski/gonja/v2"
	gonjaexec "github.com/nikolalohinski/gonja/v2/exec"
)

func ProcessJinjaTemplates(ctx context.Context, taskContent interface{}) error {
	v := reflect.ValueOf(taskContent)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if fieldType.PkgPath != "" {
			continue
		}

		vars, ok := variables.FromContext(ctx)
		if !ok {
			vars = variables.Variables{}
		}

		varsCtx := gonjaexec.NewContext(vars)

		switch field.Kind() {
		case reflect.String:
			jinjaString := field.Interface().(string)
			if jinjaString == "" {
				continue
			}
			template, err := gonja.FromString(jinjaString)
			if err != nil {
				return err
			}
			expanded, err := template.ExecuteToString(varsCtx)
			if err != nil {
				return err
			}
			field.SetString(expanded)
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					jinjaString := field.Index(j).Interface().(string)
					if jinjaString == "" {
						continue
					}
					template, err := gonja.FromString(jinjaString)
					if err != nil {
						return err
					}
					expanded, err := template.ExecuteToString(varsCtx)
					if err != nil {
						return err
					}
					field.Index(j).SetString(expanded)
				}
			}
		case reflect.Struct:
			if err := ProcessJinjaTemplates(ctx, field.Addr().Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}

func shouldApply(creates, removes string) (bool, error) {
	if creates != "" {
		matches, err := filepath.Glob(creates)
		if err != nil {
			return false, err
		}
		return len(matches) == 0, nil
	}

	if removes != "" {
		matches, err := filepath.Glob(removes)
		if err != nil {
			return false, err
		}
		return len(matches) > 0, nil
	}

	return true, nil
}

func validateCmd(argv []string, cmd, stdin string, stdinAddNewline *bool) error {
	if cmd != "" && len(argv) != 0 {
		return errors.New("cmd and argv can't be both specified at the same time")
	}

	if cmd == "" && len(argv) == 0 {
		return errors.New("either cmd or argv need to be specified")
	}

	if stdin == "" && stdinAddNewline != nil && *stdinAddNewline {
		return errors.New("stdin_add_newline can't be set if stdin is unset")
	}
	return nil
}

// applyCmd expects an *exec.Cmd that already has Args set, e.g. by calling
// exec.Command("foo").
func applyCmd(cmdFunc func() *exec.Cmd, creates, removes, chdir, stdin string, stdinAddNewline *bool) ([]byte, error) {
	ok, err := shouldApply(creates, removes)
	if err != nil {
		return []byte{}, err
	}

	if !ok {
		return []byte{}, nil
	}

	cmd := cmdFunc()

	if chdir != "" {
		cmd.Dir = chdir
	}

	if stdin != "" {
		cmdStdin := stdin

		if stdinAddNewline == nil || stdinAddNewline != nil && *stdinAddNewline {
			cmdStdin += "\n"
		}
		cmd.Stdin = strings.NewReader(cmdStdin)
	}

	return cmd.CombinedOutput()
}
