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

func TestCommandApplySuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
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

		if !cmp.Equal(expected, got) {
			t.Errorf("expected %#v but got %#v", expected, got)
		}
	})
}

func TestCommandApplyArgv(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
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

		if !cmp.Equal(expected, got) {
			t.Errorf("expected %#v but got %#v", expected, got)
		}
	})
}

func TestCommandApplyExpandArgumentVars(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
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

		if !cmp.Equal(expected, got) {
			t.Errorf("expected %#v but got %#v", expected, got)
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

		if !cmp.Equal(expected, got) {
			t.Errorf("expected %#v but got %#v", expected, got)
		}
	})
}

func TestCommandApplyError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
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

		if !cmp.Equal(expected, got) {
			t.Errorf("expected %#v but got %#v", expected, got)
		}
	})
}

func TestCommandApplyChdir(t *testing.T) {
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
	_, err := cmd.Apply(ctx, "", false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCommandApplySkipped(t *testing.T) {
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

	if !cmp.Equal(expected, got) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}
