package util

import (
	"errors"
	"os"
	"testing"
	"testing/fstest"
)

func TestNewModeFromSpec(t *testing.T) {
	fsys := fstest.MapFS{
		"hello": &fstest.MapFile{
			Mode: 0o000,
		},
		"world": &fstest.MapFile{
			Mode: 0o777,
		},
		"notevil": &fstest.MapFile{
			Mode: 0o444,
		},
	}

	tests := []struct {
		name     string
		path     string
		spec     string
		want     os.FileMode
		wantErr  bool
		checkErr func(error) bool
	}{
		{
			name: "u+rw,g=x from 000",
			path: "hello",
			spec: "u+rw,g=x",
			want: 0o610,
		},
		{
			name: "u=x,g-rw,o+w from 777",
			path: "world",
			spec: "u=x,g-rw,o+w",
			want: 0o117,
		},
		{
			name: "a=rw from 444",
			path: "notevil",
			spec: "a=rw",
			want: 0o666,
		},
		{
			name:    "invalid spec",
			path:    "hello",
			spec:    "invalid",
			wantErr: true,
			checkErr: func(err error) bool {
				return errors.Is(err, ErrInvalidMode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewModeFromSpec(fsys, tt.path, tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Error("NewModeFromSpec() expected error but got nil")
					return
				}
				if tt.checkErr != nil && !tt.checkErr(err) {
					t.Errorf("NewModeFromSpec() error = %v, failed error check", err)
				}
				return
			}
			if err != nil {
				t.Errorf("NewModeFromSpec() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("NewModeFromSpec() = %o, want %o", got, tt.want)
			}
		})
	}
}
