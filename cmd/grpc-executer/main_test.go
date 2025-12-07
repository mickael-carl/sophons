package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestTaskExecuterServer_SimpleCommand(t *testing.T) {
	logger := zap.NewNop()
	server := &taskExecuterServer{logger: logger}

	req := &proto.TaskExecuteRequest{
		Task: &proto.Task{
			Name: "test echo",
			Content: &proto.Task_Command{
				Command: &proto.Command{Cmd: "echo hello"},
			},
		},
		Variables:  &structpb.Struct{Fields: make(map[string]*structpb.Value)},
		ParentPath: "",
		IsRole:     false,
	}

	resp, err := server.TaskExecute(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskExecute returned error: %v", err)
	}

	if !resp.Success {
		t.Errorf("TaskExecute failed: %s", resp.Error)
	}

	if resp.Result == nil {
		t.Error("TaskExecute returned nil result")
	}

	// Check that stdout contains "hello"
	resultMap := resp.Result.AsMap()
	stdout, ok := resultMap["stdout"].(string)
	if !ok {
		t.Error("result does not contain stdout string")
	}
	if stdout != "hello\n" && stdout != "hello" {
		t.Errorf("unexpected stdout: got %q, want %q or %q", stdout, "hello\n", "hello")
	}
}

func TestTaskExecuterServer_WithVariables(t *testing.T) {
	logger := zap.NewNop()
	server := &taskExecuterServer{logger: logger}

	varsStruct, err := structpb.NewStruct(map[string]any{
		"message": "hello from variables",
	})
	if err != nil {
		t.Fatalf("failed to create variables struct: %v", err)
	}

	req := &proto.TaskExecuteRequest{
		Task: &proto.Task{
			Name: "test with variables",
			Content: &proto.Task_Command{
				Command: &proto.Command{Cmd: "echo {{ message }}"},
			},
		},
		Variables:  varsStruct,
		ParentPath: "",
		IsRole:     false,
	}

	resp, err := server.TaskExecute(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskExecute returned error: %v", err)
	}

	if !resp.Success {
		t.Errorf("TaskExecute failed: %s", resp.Error)
	}

	// Check that Jinja template was processed
	resultMap := resp.Result.AsMap()
	stdout, ok := resultMap["stdout"].(string)
	if !ok {
		t.Error("result does not contain stdout string")
	}
	if stdout != "hello from variables\n" && stdout != "hello from variables" {
		t.Errorf("unexpected stdout: got %q, want %q or %q", stdout, "hello from variables\n", "hello from variables")
	}
}

func TestTaskExecuterServer_WithWhenCondition(t *testing.T) {
	logger := zap.NewNop()
	server := &taskExecuterServer{logger: logger}

	req := &proto.TaskExecuteRequest{
		Task: &proto.Task{
			Name: "test with when condition",
			When: "false",
			Content: &proto.Task_Command{
				Command: &proto.Command{Cmd: "echo should not run"},
			},
		},
		Variables:  &structpb.Struct{Fields: make(map[string]*structpb.Value)},
		ParentPath: "",
		IsRole:     false,
	}

	resp, err := server.TaskExecute(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskExecute returned error: %v", err)
	}

	if !resp.Success {
		t.Errorf("TaskExecute failed: %s", resp.Error)
	}

	// Task should be skipped, so stdout should be empty
	resultMap := resp.Result.AsMap()
	stdout, _ := resultMap["stdout"].(string)
	if stdout != "" {
		t.Errorf("task should have been skipped, but stdout = %q", stdout)
	}
}

func TestTaskExecuterServer_InvalidTask(t *testing.T) {
	logger := zap.NewNop()
	server := &taskExecuterServer{logger: logger}

	// Task with invalid content (command with invalid syntax)
	req := &proto.TaskExecuteRequest{
		Task: &proto.Task{
			Name:    "test invalid task",
			Content: nil, // Invalid: no content
		},
		Variables:  &structpb.Struct{Fields: make(map[string]*structpb.Value)},
		ParentPath: "",
		IsRole:     false,
	}

	resp, err := server.TaskExecute(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskExecute returned error: %v", err)
	}

	// The task should fail during conversion
	if resp.Success {
		t.Error("TaskExecute should have failed for invalid task")
	}

	if resp.Error == "" {
		t.Error("TaskExecute should return error message for invalid task")
	}
}

func TestTaskExecuterServer_FailedCommand(t *testing.T) {
	logger := zap.NewNop()
	server := &taskExecuterServer{logger: logger}

	// Task that will fail (command not found)
	req := &proto.TaskExecuteRequest{
		Task: &proto.Task{
			Name: "test failed command",
			Content: &proto.Task_Command{
				Command: &proto.Command{Cmd: "this-command-does-not-exist"},
			},
		},
		Variables:  &structpb.Struct{Fields: make(map[string]*structpb.Value)},
		ParentPath: "",
		IsRole:     false,
	}

	resp, err := server.TaskExecute(context.Background(), req)
	if err != nil {
		t.Fatalf("TaskExecute returned error: %v", err)
	}

	// The task should fail during execution
	if resp.Success {
		t.Error("TaskExecute should have failed for command not found")
	}

	if resp.Error == "" {
		t.Error("TaskExecute should return error message for failed command")
	}
}

func TestConvertProtoStructToMap(t *testing.T) {
	tests := []struct {
		name    string
		input   *structpb.Struct
		want    map[string]any
		wantErr bool
	}{
		{
			name:    "nil struct",
			input:   nil,
			want:    map[string]any{},
			wantErr: false,
		},
		{
			name: "simple struct",
			input: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"key1": structpb.NewStringValue("value1"),
					"key2": structpb.NewNumberValue(42),
				},
			},
			want: map[string]any{
				"key1": "value1",
				"key2": 42.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertProtoStructToMap(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertProtoStructToMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("convertProtoStructToMap() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConvertMapToProtoStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    *structpb.Struct
		wantErr bool
	}{
		{
			name:    "nil map",
			input:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name: "simple map",
			input: map[string]any{
				"key1": "value1",
				"key2": 42.0,
			},
			want: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"key1": structpb.NewStringValue("value1"),
					"key2": structpb.NewNumberValue(42),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertMapToProtoStruct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertMapToProtoStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("convertMapToProtoStruct() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
