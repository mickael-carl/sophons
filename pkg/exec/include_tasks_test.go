package exec

import (
	"testing"

	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestIncludeTasksValidate(t *testing.T) {
	tests := []struct {
		name         string
		includeTasks *IncludeTasks
		wantErr      bool
		errMsg       string
	}{
		{
			name: "missing file",
			includeTasks: &IncludeTasks{
				IncludeTasks: &proto.IncludeTasks{},
			},
			wantErr: true,
			errMsg:  "`file` is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.includeTasks.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
