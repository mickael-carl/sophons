package exec

import "testing"

func TestTemplateValidate(t *testing.T) {
	tests := []struct {
		name     string
		template Template
		wantErr  bool
		errMsg   string
	}{
		{
			name: "absolute path",
			template: Template{
				Src:  "/etc/shadow",
				Dest: "/hacking-passwords",
			},
			wantErr: true,
			errMsg:  "template from an absolute path is not supported",
		},
		{
			name: "missing src",
			template: Template{
				Dest: "/something",
			},
			wantErr: true,
			errMsg:  "src is required",
		},
		{
			name: "missing dest",
			template: Template{
				Src: "foo",
			},
			wantErr: true,
			errMsg:  "dest is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.template.Validate()
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
