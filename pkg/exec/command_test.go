package exec

import (
	"context"
	"errors"
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/mock/gomock"
)

type testExitError struct {
	code int
	err  error
}

func (e *testExitError) Error() string {
	return e.err.Error()
}

func (e *testExitError) ExitCode() int {
	return e.code
}

func TestCommandApply(t *testing.T) {
	tests := []struct {
		name       string
		useSyncTest bool
		testFunc   func(*testing.T)
	}{
		{
			name:       "success",
			useSyncTest: true,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				cmd := &Command{
					Cmd: "/bin/ls /",
				}

				start := time.Now()

				var stdout io.Writer
				mockExecutor.EXPECT().SetStdout(gomock.Any()).Do(func(w io.Writer) { stdout = w })
				mockExecutor.EXPECT().SetStderr(gomock.Any())
				mockExecutor.EXPECT().Run().DoAndReturn(func() error {
					stdout.Write([]byte("bin boot dev etc home lib media mnt opt proc root run sbin srv sys tmp usr var"))
					time.Sleep(1 * time.Second)
					return nil
				})

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)

				got, err := cmd.Apply(ctx, "", false)
				if err != nil {
					t.Error(err)
				}

				expected := &CommandResult{
					CommonResult: CommonResult{
						Changed: true,
						Stdout:  "bin boot dev etc home lib media mnt opt proc root run sbin srv sys tmp usr var",
					},
					Cmd:   []string{"/bin/ls", "/"},
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
			name:       "argv",
			useSyncTest: true,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				cmd := &Command{
					Argv: []string{"/bin/ls", "/"},
				}

				var stdout io.Writer
				mockExecutor.EXPECT().SetStdout(gomock.Any()).Do(func(w io.Writer) { stdout = w })
				mockExecutor.EXPECT().SetStderr(gomock.Any())
				mockExecutor.EXPECT().Run().DoAndReturn(func() error {
					stdout.Write([]byte("bin boot dev etc home lib media mnt opt proc root run sbin srv sys tmp usr var"))
					return nil
				})

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)

				got, err := cmd.Apply(ctx, "", false)
				if err != nil {
					t.Error(err)
				}

				expected := &CommandResult{
					CommonResult: CommonResult{
						Changed: true,
						Stdout:  "bin boot dev etc home lib media mnt opt proc root run sbin srv sys tmp usr var",
					},
					Cmd:   []string{"/bin/ls", "/"},
					End:   time.Now(),
					Start: time.Now(),
				}

				if diff := cmp.Diff(expected, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:       "expand argument vars",
			useSyncTest: true,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				t.Setenv("ROOT", "/")

				cmd := &Command{
					Argv:               []string{"/bin/ls", "$ROOT"},
					ExpandArgumentVars: true,
				}

				var stdout io.Writer
				mockExecutor.EXPECT().SetStdout(gomock.Any()).Do(func(w io.Writer) { stdout = w })
				mockExecutor.EXPECT().SetStderr(gomock.Any())
				mockExecutor.EXPECT().Run().DoAndReturn(func() error {
					stdout.Write([]byte("bin boot dev etc home lib media mnt opt proc root run sbin srv sys tmp usr var"))
					return nil
				})

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)

				got, err := cmd.Apply(ctx, "", false)
				if err != nil {
					t.Error(err)
				}

				expected := &CommandResult{
					CommonResult: CommonResult{
						Changed: true,
						Stdout:  "bin boot dev etc home lib media mnt opt proc root run sbin srv sys tmp usr var",
					},
					Cmd:   []string{"/bin/ls", "/"},
					End:   time.Now(),
					Start: time.Now(),
				}

				if diff := cmp.Diff(expected, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}

				cmd = &Command{
					Cmd:                "/bin/ls $ROOT",
					ExpandArgumentVars: true,
				}

				mockExecutor.EXPECT().SetStdout(gomock.Any()).Do(func(w io.Writer) { stdout = w })
				mockExecutor.EXPECT().SetStderr(gomock.Any())
				mockExecutor.EXPECT().Run().DoAndReturn(func() error {
					stdout.Write([]byte("bin boot dev etc home lib media mnt opt proc root run sbin srv sys tmp usr var"))
					return nil
				})

				ctx = context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)

				got, err = cmd.Apply(ctx, "", false)
				if err != nil {
					t.Error(err)
				}

				expected = &CommandResult{
					CommonResult: CommonResult{
						Changed: true,
						Stdout:  "bin boot dev etc home lib media mnt opt proc root run sbin srv sys tmp usr var",
					},
					Cmd:   []string{"/bin/ls", "/"},
					End:   time.Now(),
					Start: time.Now(),
				}

				if diff := cmp.Diff(expected, got); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:       "error",
			useSyncTest: true,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				cmd := &Command{
					Cmd: "ls /foo",
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

				expected := &CommandResult{
					CommonResult: CommonResult{
						Failed: true,
						RC:     2,
						Stderr: "ls: /foo: No such file or directory",
					},
					Cmd:   []string{"ls", "/foo"},
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
			name:       "chdir",
			useSyncTest: false,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				cmd := &Command{
					Cmd:   "ls",
					Chdir: "/tmp",
				}

				mockExecutor.EXPECT().SetDir("/tmp")
				mockExecutor.EXPECT().SetStdout(gomock.Any())
				mockExecutor.EXPECT().SetStderr(gomock.Any())
				mockExecutor.EXPECT().Run().Return(nil)

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)
				got, err := cmd.Apply(ctx, "", false)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}

				expected := &CommandResult{
					CommonResult: CommonResult{
						Changed: true,
					},
					Cmd: []string{"ls"},
				}

				// Ignore time fields since we're not using synctest
				opts := cmp.Options{
					cmp.FilterPath(func(p cmp.Path) bool {
						return p.String() == "Start" || p.String() == "End" || p.String() == "Delta"
					}, cmp.Ignore()),
				}

				if diff := cmp.Diff(expected, got, opts); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:       "skipped",
			useSyncTest: false,
			testFunc: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockExecutor := NewMockcommandExecutor(ctrl)
				mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
					return mockExecutor
				})

				cmd := &Command{
					Cmd:     "rm /foo",
					Removes: "/foo",
				}

				ctx := context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)

				got, err := cmd.Apply(ctx, "", false)
				if err != nil {
					t.Error(err)
				}

				expected := &CommandResult{
					CommonResult: CommonResult{
						Skipped: true,
					},
					Cmd: []string{"rm", "/foo"},
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
