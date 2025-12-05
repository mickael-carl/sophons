package proto

import (
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mickael-carl/sophons/pkg/registry"
)

func init() {
	yaml.RegisterCustomUnmarshaler[[]*Task](tasksUnmarshalYAML)
}

func tasksUnmarshalYAML(t *[]*Task, b []byte) error {
	type unmarshalTask struct {
		Name       string              `yaml:"name"`
		When       string              `yaml:"when"`
		Loop       any                 `yaml:"loop"`
		Register   string              `yaml:"register"`
		RawContent map[string]ast.Node `yaml:",inline"`
	}

	var raw []unmarshalTask
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return err
	}

	var tasksOut []*Task
	for _, task := range raw {
		protoTask := &Task{
			Name:     task.Name,
			When:     task.When,
			Register: task.Register,
		}

		if task.Loop != nil {
			loopValue, err := structpb.NewValue(task.Loop)
			if err != nil {
				return fmt.Errorf("failed to convert loop to structpb.Value: %w", err)
			}
			protoTask.Loop = loopValue
		}

		for moduleName, node := range task.RawContent {
			reg, ok := registry.NameRegistry[moduleName]
			if !ok {
				continue
			}

			protoMsg := reg.ProtoFactory()
			if err := yaml.NodeToValue(node, protoMsg); err != nil {
				return fmt.Errorf("failed to unmarshal %s module: %w", moduleName, err)
			}

			protoTask.Content = reg.ProtoWrapper(protoMsg).(isTask_Content)
			break
		}

		if protoTask.Content == nil {
			return fmt.Errorf("task %q has no recognized module content", task.Name)
		}

		tasksOut = append(tasksOut, protoTask)
	}

	*t = tasksOut
	return nil
}
