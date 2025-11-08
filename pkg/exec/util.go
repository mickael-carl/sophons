package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mickael-carl/sophons/pkg/variables"
	"github.com/nikolalohinski/gonja/v2"
	gonjaexec "github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
	"github.com/nikolalohinski/gonja/v2/nodes"
)

func ProcessJinjaTemplates(ctx context.Context, taskContent interface{}) error {
	v := reflect.ValueOf(taskContent)
	for v.Kind() == reflect.Ptr {
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
			jinjaString := field.String()
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
					jinjaString := field.Index(j).String()
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

		case reflect.Interface:
			if field.IsNil() {
				continue
			}

			// If the interface we're getting is a string, let's treat it as a
			// Jinja template that needs rendering. The output might be a
			// string, but not only. For now we also see a case where the
			// output could be a list so let's support that by parsing the
			// actual output.
			if str, ok := field.Interface().(string); ok {
				// Ignore anything that's not a Jinja template.
				if !strings.Contains(str, "{{") {
					continue
				}

				template, err := gonja.FromString(str)
				if err != nil {
					return err
				}

				loader, err := loaders.NewMemoryLoader(map[string]string{})
				if err != nil {
					return err
				}

				env := gonja.DefaultEnvironment
				env.Context = varsCtx

				var buf bytes.Buffer
				renderer := gonjaexec.NewRenderer(env, &buf, gonja.DefaultConfig, loader, template)
				// This is the slightly complex part: we want to evaluate the
				// template node, which is the first one under the root node.
				// To do that we need an expression, which is a field in output
				// nodes, so we cast our first node (basically the jinja
				// variable) into such a node. See
				// https://github.com/nikolalohinski/gonja/blob/v2.4.0/nodes/nodes.go#L12-L27.
				value := renderer.Eval(renderer.RootNode.Nodes[0].(*nodes.Output).Expression)

				// If it's a list, `value` contains effectively a
				// `[]interface{}` so we need to make a `[]string` out of that.
				if value.IsList() {
					outSlice := []string{}
					for _, i := range value.ToGoSimpleType(false).([]interface{}) {
						s, ok := i.(string)
						if !ok {
							return fmt.Errorf("found a non-string value in a jinja list, which is not supported")
						}
						outSlice = append(outSlice, s)
					}

					field.Set(reflect.ValueOf(outSlice))
					continue
				}

				// Not a list means we can assume it's a simple type and
				// doesn't need special handling.
				field.Set(reflect.ValueOf(value.ToGoSimpleType(false)))
				continue
			}

			// If we're dealing with a slice, we need to iterate over its
			// elements and process them.
			if reflect.TypeOf(field.Interface()).Kind() == reflect.Slice {
				s := reflect.ValueOf(field.Interface())

				for i := 0; i < s.Len(); i++ {
					elem := s.Index(i)
					if elem.Kind() == reflect.String {
						jinjaString := elem.String()
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
						elem.SetString(expanded)
					}
				}
				continue
			}

			if field.Elem().Kind() == reflect.Ptr {
				if err := ProcessJinjaTemplates(ctx, field.Elem().Interface()); err != nil {
					return err
				}
			} else {
				newValue := reflect.New(field.Elem().Type())
				newValue.Elem().Set(field.Elem())
				if err := ProcessJinjaTemplates(ctx, newValue.Interface()); err != nil {
					return err
				}
				field.Set(newValue)
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

func getStringSlice(i interface{}) []string {
	if i == nil {
		return nil
	}
	if str, ok := i.(string); ok {
		return []string{str}
	}
	if slice, ok := i.([]string); ok {
		return slice
	}
	return nil
}
