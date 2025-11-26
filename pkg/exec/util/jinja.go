package util

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/nikolalohinski/gonja/v2"
	gonjaexec "github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
	"github.com/nikolalohinski/gonja/v2/nodes"

	"github.com/mickael-carl/sophons/pkg/variables"
)

func JinjaProcessWhen(ctx context.Context, when string) (bool, error) {
	if when == "" {
		return true, nil
	}

	vars, ok := variables.FromContext(ctx)
	if !ok {
		vars = variables.Variables{}
	}
	varsCtx := gonjaexec.NewContext(vars)

	template, err := gonja.FromString("{{ " + when + " }}")
	if err != nil {
		return false, err
	}

	result, err := template.ExecuteToString(varsCtx)
	if err != nil {
		return false, err
	}

	// TODO: For now, we'll consider "true" as true, and anything else as
	// false. Ansible has more complex rules, but this is a start.
	// https://docs.ansible.com/ansible/latest/user_guide/playbooks_conditionals.html#conditionals
	switch result {
	case "True", "true":
		return true, nil
	case "False", "false":
		return false, nil
	}

	// Attempt to convert to a number
	if i, err := strconv.Atoi(result); err == nil {
		return i != 0, nil
	}

	return false, nil
}

// renderJinjaStringToSlice renders a Jinja template string and returns a []string.
// If the template evaluates to a list, all elements are returned.
// If the template evaluates to a single value, a single-element slice is returned.
// If the string is empty or doesn't contain "{{", it returns the original string.
func renderJinjaStringToSlice(jinjaString string, varsCtx *gonjaexec.Context) ([]string, error) {
	if jinjaString == "" {
		return []string{""}, nil
	}

	// Not a Jinja template, return as-is
	if !strings.Contains(jinjaString, "{{") {
		return []string{jinjaString}, nil
	}

	template, err := gonja.FromString(jinjaString)
	if err != nil {
		return nil, err
	}

	loader, err := loaders.NewMemoryLoader(map[string]string{})
	if err != nil {
		return nil, err
	}

	env := gonja.DefaultEnvironment
	env.Context = varsCtx

	var buf bytes.Buffer
	renderer := gonjaexec.NewRenderer(env, &buf, gonja.DefaultConfig, loader, template)

	// Special case: if the template is ONLY a variable reference (e.g., "{{ foo }}")
	// and it evaluates to a list, we want to expand it. Otherwise, we just render
	// the template as a string.
	if len(renderer.RootNode.Nodes) == 1 {
		if outputNode, ok := renderer.RootNode.Nodes[0].(*nodes.Output); ok {
			value := renderer.Eval(outputNode.Expression)

			if value.IsNil() {
				return nil, nil
			}

			// If it's a list, convert to []string
			if value.IsList() {
				outSlice := []string{}
				for _, i := range value.ToGoSimpleType(false).([]any) {
					s, ok := i.(string)
					if !ok {
						return nil, fmt.Errorf("found a non-string value in a jinja list, which is not supported")
					}
					outSlice = append(outSlice, s)
				}
				return outSlice, nil
			}
		}
	}

	// For mixed templates (e.g., "prefix-{{ var }}"), just render as a string
	rendered, err := template.ExecuteToString(varsCtx)
	if err != nil {
		return nil, err
	}

	return []string{rendered}, nil
}

func ProcessJinjaTemplates(ctx context.Context, taskContent any) error {
	v := reflect.ValueOf(taskContent)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
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
				// Need to handle the case where a template might expand to a list
				newSlice := []string{}
				for j := 0; j < field.Len(); j++ {
					jinjaString := field.Index(j).String()

					rendered, err := renderJinjaStringToSlice(jinjaString, varsCtx)
					if err != nil {
						return err
					}

					// rendered is nil when the value is nil, skip it
					if rendered != nil {
						newSlice = append(newSlice, rendered...)
					}
				}

				// Replace the slice with the new one
				field.Set(reflect.ValueOf(newSlice))
			}

		case reflect.Struct:
			if err := ProcessJinjaTemplates(ctx, field.Addr().Interface()); err != nil {
				return err
			}

		case reflect.Ptr:
			if field.IsNil() {
				continue
			}
			// Get the element the pointer points to.
			elem := field.Elem()

			// Recursively process the element pointed to by the pointer.
			switch elem.Kind() {
			case reflect.Struct:
				// Pass a pointer to the element to `ProcessJinjaTemplates` for struct processing.
				if err := ProcessJinjaTemplates(ctx, elem.Addr().Interface()); err != nil {
					return err
				}
			case reflect.Slice:
				if elem.Type().Elem().Kind() == reflect.String {
					// Need to handle the case where a template might expand to a list
					newSlice := []string{}
					for j := 0; j < elem.Len(); j++ {
						stringElem := elem.Index(j)
						jinjaString := stringElem.String()

						rendered, err := renderJinjaStringToSlice(jinjaString, varsCtx)
						if err != nil {
							return err
						}

						if rendered != nil {
							newSlice = append(newSlice, rendered...)
						}
					}

					elem.Set(reflect.ValueOf(newSlice))
				}
			case reflect.String:
				if elem.CanSet() {
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
				rendered, err := renderJinjaStringToSlice(str, varsCtx)
				if err != nil {
					return err
				}

				// rendered is nil when the value is nil
				if rendered == nil {
					continue
				}

				// If it's a list (more than one element), set as []string
				if len(rendered) > 1 {
					field.Set(reflect.ValueOf(rendered))
					continue
				}

				// Single element - set as the unwrapped string
				if len(rendered) == 1 {
					field.Set(reflect.ValueOf(rendered[0]))
					continue
				}
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
