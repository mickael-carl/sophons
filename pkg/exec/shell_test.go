package exec

import (
	"context"
	"errors"
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mickael-carl/sophons/pkg/proto"
	"go.uber.org/mock/gomock"
)

func TestShellApply(t *testing.T) {
	tests := []struct {
		name        string
		useSyncTest bool
		testFunc    func(*testing.T)
	}{
		{
			name:        "success",
			useSyncTest: true,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				s := &Shell{
					Shell: proto.Shell{
						Cmd: "echo hello",
					},
				}

				start := time.Now()

				var stdout io.Writer
				mockExecutor.EXPECT().SetStdout(gomock.Any()).Do(func(w io.Writer) { stdout = w })
				mockExecutor.EXPECT().SetStderr(gomock.Any())
				mockExecutor.EXPECT().Run().DoAndReturn(func() error {
					stdout.Write([]byte("hello"))
					time.Sleep(1 * time.Second)
					return nil
				})

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)

				got, err := s.Apply(ctx, "", false)
				if err != nil {
					t.Error(err)
				}

				expected := &ShellResult{
					CommonResult: CommonResult{
						Changed: true,
						Stdout:  "hello",
					},
					Cmd:   "echo hello",
					Delta: time.Second,
					End:   start.Add(1 * time.Second),
					Start: start,
				}

				if diff := cmp.Diff(expected, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:        "error",
			useSyncTest: true,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				cmd := &Shell{
					Shell: proto.Shell{
						Cmd: "ls /foo",
					},
				}

				var stderr io.Writer
				mockExecutor.EXPECT().SetStdout(gomock.Any())
				mockExecutor.EXPECT().SetStderr(gomock.Any()).Do(func(w io.Writer) { stderr = w })
				mockExecutor.EXPECT().Run().DoAndReturn(func() error {
					stderr.Write([]byte("ls: /foo: No such file or directory"))
					return &testExitError{2, errors.New("exit status 2")}
				})

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)

				got, err := cmd.Apply(ctx, "", false)

				if err == nil {
					t.Fatal("expected an error but got none")
				}

				if err.Error() != "failed to execute command: exit status 2" {
					t.Error(err)
				}

				expected := &ShellResult{
					CommonResult: CommonResult{
						Failed: true,
						RC:     2,
						Stderr: "ls: /foo: No such file or directory",
					},
					Cmd:   "ls /foo",
					Delta: 0,
					End:   time.Now(),
					Start: time.Now(),
				}

				if diff := cmp.Diff(expected, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:        "full options",
			useSyncTest: true,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				cmd := &Shell{
					Shell: proto.Shell{
						Argv:       []string{"echo", "hello"},
						Chdir:      "/tmp",
						Executable: "/usr/bin/fish",
					},
				}

				start := time.Now()

				var stdout io.Writer
				mockExecutor.EXPECT().SetDir("/tmp")
				mockExecutor.EXPECT().SetStdout(gomock.Any()).Do(func(w io.Writer) { stdout = w })
				mockExecutor.EXPECT().SetStderr(gomock.Any())
				mockExecutor.EXPECT().Run().DoAndReturn(func() error {
					stdout.Write([]byte("hello"))
					time.Sleep(1 * time.Second)
					return nil
				})

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)
				got, err := cmd.Apply(ctx, "", false)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				expected := &ShellResult{
					CommonResult: CommonResult{
						Changed: true,
						Stdout:  "hello",
					},
					Cmd:   "echo hello",
					Delta: time.Second,
					End:   start.Add(time.Second),
					Start: start,
				}

				if diff := cmp.Diff(expected, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:        "skipped",
			useSyncTest: false,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				s := &Shell{
					Shell: proto.Shell{
						Cmd:     "rm /foo",
						Removes: "/foo",
					},
				}

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)

				got, err := s.Apply(ctx, "", false)
				if err != nil {
					t.Error(err)
				}

				expected := &ShellResult{
					CommonResult: CommonResult{
						Skipped: true,
					},
					Cmd: "rm /foo",
				}

				if diff := cmp.Diff(expected, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.useSyncTest {
				synctest.Test(t, tt.testFunc)
			} else {
				tt.testFunc(t)
			}
		})
	}
}
