package exec

import (
	"context"
	"testing"
	"testing/fstest"

	"go.uber.org/mock/gomock"
)

// newMockAptContext creates a test context with a mocked apt client.
// The setupFunc is called with the mock to configure expectations.
func newMockAptContext(t *testing.T, setupFunc func(*MockaptClient)) context.Context {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	m := NewMockaptClient(ctrl)
	if setupFunc != nil {
		setupFunc(m)
	}

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	return context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})
}

// newMockCommandContext creates a test context with a mocked command executor.
// The setupFunc is called with the mock to configure expectations.
func newMockCommandContext(t *testing.T, setupFunc func(*MockcommandExecutor)) context.Context {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockExecutor := NewMockcommandExecutor(ctrl)
	mockCmdFactory := cmdFactory(func(name string, args ...string) commandExecutor {
		return mockExecutor
	})

	if setupFunc != nil {
		setupFunc(mockExecutor)
	}

	return context.WithValue(context.Background(), commandFactoryContextKey, mockCmdFactory)
}

// ValidationTestCase represents a test case for validation functions.
type ValidationTestCase[T TaskContent] struct {
	Name        string
	Input       T
	WantErr     bool
	ErrMsg      string
	ErrContains string // Optional: check if error contains this substring
}

// RunValidationTests runs a table of validation tests.
func RunValidationTests[T TaskContent](t *testing.T, tests []ValidationTestCase[T]) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			err := tt.Input.Validate()
			if (err != nil) != tt.WantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.WantErr)
				return
			}
			if tt.WantErr && err != nil {
				if tt.ErrMsg != "" && err.Error() != tt.ErrMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.ErrMsg)
				}
				if tt.ErrContains != "" && !contains(err.Error(), tt.ErrContains) {
					t.Errorf("Validate() error = %q, should contain %q", err.Error(), tt.ErrContains)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
