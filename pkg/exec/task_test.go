package exec

import (
	"context"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/arduino/go-apt-client"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/registry"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestDeepCopyContent(t *testing.T) {
	originalContent := &Command{
		Command: &proto.Command{
			Cmd:   "echo hello",
			Chdir: "/tmp",
			Argv:  []string{},
		},
	}

	copiedContent, err := deepCopyContent(originalContent)
	if err != nil {
		t.Fatalf("deepCopyContent failed: %v", err)
	}

	if originalContent == copiedContent {
		t.Error("copied content is the same instance as original")
	}

	if diff := cmp.Diff(originalContent, copiedContent, cmpopts.IgnoreUnexported(Command{}, proto.Command{}), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	copiedCommand, ok := copiedContent.(*Command)
	if !ok {
		t.Error("copied content is not of type *Command")
	}
	copiedCommand.Cmd = "echo world"
	copiedCommand.Chdir = "/var"

	if originalContent.Cmd != "echo hello" {
		t.Errorf("original content's Cmd changed to %q, expected %q", originalContent.Cmd, "echo hello")
	}
	if originalContent.Chdir != "/tmp" {
		t.Errorf("original content's Chdir changed to %q, expected %q", originalContent.Chdir, "/tmp")
	}

	if copiedCommand.Cmd != "echo world" {
		t.Errorf("copied content's Cmd is %q, expected %q", copiedCommand.Cmd, "echo world")
	}
	if copiedCommand.Chdir != "/var" {
		t.Errorf("copied content's Chdir is %q, expected %q", copiedCommand.Chdir, "/var")
	}
}

func TestTaskApply(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	task := Task{
		Name: "Add docker repo",
		Content: &AptRepository{
			AptRepository: &proto.AptRepository{
				Repo:  "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
				State: AptRepositoryPresent,
			},
		},
		Register: "out",
	}

	repo := &apt.Repository{
		Enabled:      true,
		SourceRepo:   false,
		Options:      "signed-by=/etc/apt/keyrings/docker.asc",
		URI:          "https://download.docker.com/linux/debian",
		Distribution: "bookworm",
		Components:   "stable",
		Comment:      "",
	}

	m.EXPECT().ParseAPTConfigFolder("/etc/apt").Return(nil, nil)
	m.EXPECT().ParseAPTConfigLine("deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable").Return(repo)
	m.EXPECT().AddRepository(repo, "/etc/apt", "download_docker_com_linux_debian.list").Return(nil)
	m.EXPECT().CheckForUpdates().Return("", nil)

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = variables.NewContext(ctx, variables.Variables{})
	logger := zap.NewNop()
	if err := ExecuteTask(ctx, logger, task, "", false); err != nil {
		t.Error(err)
	}

	vars, ok := variables.FromContext(ctx)
	if !ok {
		t.Error("failed to get variables from context")
	}

	got, ok := vars["out"]
	if !ok {
		t.Error("result is not registered in variables")
	}

	expected := map[string]any{
		"changed":      true,
		"failed":       false,
		"msg":          "",
		"rc":           uint64(0),
		"skipped":      false,
		"stderr":       "",
		"stderr_lines": []any{},
		"stdout":       "",
		"stdout_lines": []any{},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestTaskApplyLoop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	task := Task{
		Name: "install foo and {{ bar }}",
		When: "hello",
		Loop: []string{
			"foo",
			"{{ bar }}",
		},
		Content: &Apt{
			Apt: &proto.Apt{
				Name: &proto.PackageList{
					Items: []string{"{{ item }}"},
				},
			},
		},
		Register: "out",
	}

	m.EXPECT().ListInstalled().Return(nil, nil)
	m.EXPECT().Install(&apt.Package{Name: "foo"}).Return("", nil)
	m.EXPECT().ListInstalled().Return(nil, nil)
	m.EXPECT().Install(&apt.Package{Name: "bar"}).Return("", nil)

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})
	ctx = variables.NewContext(ctx, variables.Variables{
		"bar": "bar",
	})

	logger := zap.NewNop()
	if err := ExecuteTask(ctx, logger, task, "", false); err != nil {
		t.Error(err)
	}

	vars, ok := variables.FromContext(ctx)
	if !ok {
		t.Error("failed to get variables from context")
	}

	got, ok := vars["out"]
	if !ok {
		t.Error("result is not registered in variables")
	}

	expected := map[string]any{
		"changed": bool(true),
		"failed":  bool(false),
		"msg":     string(""),
		"results": []any{
			map[string]any{
				"cache_update_time": "1970-01-01T00:00:00Z",
				"cache_updated":     false,
				"changed":           true,
				"failed":            false,
				"msg":               "",
				"rc":                uint64(0),
				"skipped":           false,
				"stderr":            "",
				"stderr_lines":      []any{},
				"stdout":            "",
				"stdout_lines":      []any{},
			},
			map[string]any{
				"cache_update_time": "1970-01-01T00:00:00Z",
				"cache_updated":     false,
				"changed":           true,
				"failed":            false,
				"msg":               "",
				"rc":                uint64(0),
				"skipped":           false,
				"stderr":            "",
				"stderr_lines":      []any{},
				"stdout":            "",
				"stdout_lines":      []any{},
			},
		},
		"rc":           uint64(0),
		"skipped":      false,
		"stderr":       "",
		"stderr_lines": []any{},
		"stdout":       "",
		"stdout_lines": []any{},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFromProto(t *testing.T) {
	tests := []struct {
		name    string
		pt      *proto.Task
		setup   func()
		cleanup func()
		want    *Task
		wantErr string
	}{
		{
			name: "basic task with all fields",
			pt: &proto.Task{
				Name:     "test task",
				When:     "true",
				Register: "result",
				Content: &proto.Task_Command{
					Command: &proto.Command{
						Cmd: "echo hello",
					},
				},
			},
			want: &Task{
				Name:     "test task",
				When:     "true",
				Register: "result",
				Content: &Command{
					Command: &proto.Command{
						Cmd: "echo hello",
					},
				},
			},
		},
		{
			name: "task with loop",
			pt: &proto.Task{
				Name: "test with loop",
				Loop: &structpb.Value{
					Kind: &structpb.Value_ListValue{
						ListValue: &structpb.ListValue{
							Values: []*structpb.Value{
								{Kind: &structpb.Value_StringValue{StringValue: "item1"}},
								{Kind: &structpb.Value_StringValue{StringValue: "item2"}},
							},
						},
					},
				},
				Content: &proto.Task_Command{
					Command: &proto.Command{
						Cmd: "echo {{ item }}",
					},
				},
			},
			want: &Task{
				Name: "test with loop",
				Loop: []any{"item1", "item2"},
				Content: &Command{
					Command: &proto.Command{
						Cmd: "echo {{ item }}",
					},
				},
			},
		},
		{
			name: "task with nil content",
			pt: &proto.Task{
				Name: "task without content",
				When: "inventory_hostname == 'localhost'",
			},
			want: &Task{
				Name:    "task without content",
				When:    "inventory_hostname == 'localhost'",
				Content: nil,
			},
		},
		{
			name: "task with nil loop",
			pt: &proto.Task{
				Name: "task without loop",
				Loop: nil,
				Content: &proto.Task_Shell{
					Shell: &proto.Shell{
						Cmd: "ls -la",
					},
				},
			},
			want: &Task{
				Name: "task without loop",
				Loop: nil,
				Content: &Shell{
					Shell: &proto.Shell{
						Cmd: "ls -la",
					},
				},
			},
		},
		{
			name: "error: unregistered content type",
			pt: &proto.Task{
				Name: "task with unknown type",
				Content: &proto.Task_Command{
					Command: &proto.Command{
						Cmd: "echo test",
					},
				},
			},
			setup: func() {
				// Temporarily remove the command registration
				delete(registry.TypeRegistry, reflect.TypeFor[*proto.Task_Command]())
			},
			cleanup: func() {
				// Restore the command registration
				reg := registry.TaskRegistration{
					ProtoFactory: func() any { return &proto.Command{} },
					ProtoWrapper: func(msg any) any { return &proto.Task_Command{Command: msg.(*proto.Command)} },
					ExecAdapter: func(content any) any {
						if c, ok := content.(*proto.Task_Command); ok {
							return &Command{Command: c.Command}
						}
						return nil
					},
				}
				registry.Register("command", reg, (*proto.Task_Command)(nil))
				registry.Register("ansible.builtin.command", reg, (*proto.Task_Command)(nil))
			},
			wantErr: "unknown proto content type *proto.Task_Command: not registered",
		},
		{
			name: "error: no exec adapter",
			pt: &proto.Task{
				Name: "task with no adapter",
				Content: &proto.Task_Shell{
					Shell: &proto.Shell{
						Cmd: "test",
					},
				},
			},
			setup: func() {
				// Register a type without ExecAdapter
				reg := registry.TaskRegistration{
					ProtoFactory: func() any { return &proto.Shell{} },
					ProtoWrapper: func(msg any) any { return &proto.Task_Shell{Shell: msg.(*proto.Shell)} },
					ExecAdapter:  nil, // No adapter
				}
				registry.TypeRegistry[reflect.TypeFor[*proto.Task_Shell]()] = reg
			},
			cleanup: func() {
				// Restore the shell registration
				reg := registry.TaskRegistration{
					ProtoFactory: func() any { return &proto.Shell{} },
					ProtoWrapper: func(msg any) any { return &proto.Task_Shell{Shell: msg.(*proto.Shell)} },
					ExecAdapter: func(content any) any {
						if s, ok := content.(*proto.Task_Shell); ok {
							return &Shell{Shell: s.Shell}
						}
						return nil
					},
				}
				registry.Register("shell", reg, (*proto.Task_Shell)(nil))
				registry.Register("ansible.builtin.shell", reg, (*proto.Task_Shell)(nil))
			},
			wantErr: "no exec adapter registered for type *proto.Task_Shell",
		},
		{
			name: "error: adapter returns non-TaskContent",
			pt: &proto.Task{
				Name: "task with bad adapter",
				Content: &proto.Task_Copy{
					Copy: &proto.Copy{
						Src:  "source.txt",
						Dest: "dest.txt",
					},
				},
			},
			setup: func() {
				// Register an adapter that returns a non-TaskContent type
				reg := registry.TaskRegistration{
					ProtoFactory: func() any { return &proto.Copy{} },
					ProtoWrapper: func(msg any) any { return &proto.Task_Copy{Copy: msg.(*proto.Copy)} },
					ExecAdapter: func(content any) any {
						// Return a string instead of TaskContent
						return "not a TaskContent"
					},
				}
				registry.TypeRegistry[reflect.TypeFor[*proto.Task_Copy]()] = reg
			},
			cleanup: func() {
				// Restore the copy registration
				reg := registry.TaskRegistration{
					ProtoFactory: func() any { return &proto.Copy{} },
					ProtoWrapper: func(msg any) any { return &proto.Task_Copy{Copy: msg.(*proto.Copy)} },
					ExecAdapter: func(content any) any {
						if c, ok := content.(*proto.Task_Copy); ok {
							return &Copy{Copy: c.Copy}
						}
						return nil
					},
				}
				registry.Register("copy", reg, (*proto.Task_Copy)(nil))
				registry.Register("ansible.builtin.copy", reg, (*proto.Task_Copy)(nil))
			},
			wantErr: "exec adapter returned non-TaskContent type string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			got, err := FromProto(tt.pt)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("FromProto() error = nil, wantErr %q", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr {
					t.Errorf("FromProto() error = %q, wantErr %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("FromProto() unexpected error = %v", err)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreUnexported(
				Task{},
				Command{},
				Shell{},
				proto.Command{},
				proto.Shell{},
			)); diff != "" {
				t.Errorf("FromProto() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
