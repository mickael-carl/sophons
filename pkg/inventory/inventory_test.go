package inventory

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want Inventory
	}{
		{
			name: "complex inventory structure",
			yaml: `
usa:
  children:
    southeast:
      children:
        atlanta:
          hosts:
            host1:
            host2:
        raleigh:
          hosts:
            host2:
            host3:
      vars:
        some_server: foo.southeast.example.com
        halon_system_timeout: 30
        self_destruct_countdown: 60
        escape_pods: 2
    northeast:
    northwest:
    southwest:
`,
			want: Inventory{
				Groups: map[string]Group{
					"usa": {
						Children: map[string]Group{
							"northeast": {},
							"northwest": {},
							"southwest": {},
							"southeast": {
								Vars: map[string]any{
									"some_server":             "foo.southeast.example.com",
									"halon_system_timeout":    uint64(30),
									"self_destruct_countdown": uint64(60),
									"escape_pods":             uint64(2),
								},
								Children: map[string]Group{
									"atlanta": {
										Hosts: map[string]variables.Variables{
											"host1": nil,
											"host2": nil,
										},
									},
									"raleigh": {
										Hosts: map[string]variables.Variables{
											"host2": nil,
											"host3": nil,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Inventory
			if err := yaml.Unmarshal([]byte(tt.yaml), &got); err != nil {
				t.Errorf("Unmarshal() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupFind(t *testing.T) {
	tests := []struct {
		name     string
		group    Group
		topName  string
		hostName string
		want     map[string]struct{}
	}{
		{
			name: "simple",
			group: Group{
				Hosts: map[string]variables.Variables{
					"foo": {},
					"bar": {},
					"baz": {},
				},
			},
			topName:  "top",
			hostName: "bar",
			want:     map[string]struct{}{"top": {}},
		},
		{
			name: "nested",
			group: Group{
				Hosts: map[string]variables.Variables{
					"foo": {},
				},
				Children: map[string]Group{
					"fruit": {
						Hosts: map[string]variables.Variables{
							"myBanana": {},
						},
						Children: map[string]Group{
							"apple": {
								Hosts: map[string]variables.Variables{
									"myGala": {},
								},
							},
						},
					},
				},
			},
			topName:  "top",
			hostName: "myBanana",
			want:     map[string]struct{}{"fruit": {}, "top": {}},
		},
		{
			name: "deeply nested",
			group: Group{
				Hosts: map[string]variables.Variables{
					"foo": {},
				},
				Children: map[string]Group{
					"fruit": {
						Hosts: map[string]variables.Variables{
							"myBanana": {},
						},
						Children: map[string]Group{
							"apple": {
								Hosts: map[string]variables.Variables{
									"myGala": {},
								},
							},
						},
					},
				},
			},
			topName:  "top",
			hostName: "myGala",
			want:     map[string]struct{}{"apple": {}, "fruit": {}, "top": {}},
		},
		{
			name: "in multiple sets",
			group: Group{
				Hosts: map[string]variables.Variables{
					"foo": {},
				},
				Children: map[string]Group{
					"fruit": {
						Hosts: map[string]variables.Variables{
							"myBanana": {},
						},
						Children: map[string]Group{
							"apple": {
								Hosts: map[string]variables.Variables{
									"myGala": {},
								},
							},
						},
					},
					"party": {
						Hosts: map[string]variables.Variables{
							"myGala": {},
						},
					},
				},
			},
			topName:  "top",
			hostName: "myGala",
			want:     map[string]struct{}{"apple": {}, "fruit": {}, "party": {}, "top": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.group.Find(tt.topName, tt.hostName)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInventoryFind(t *testing.T) {
	tests := []struct {
		name      string
		inventory Inventory
		hostName  string
		want      map[string]struct{}
	}{
		{
			name: "simple",
			inventory: Inventory{
				Groups: map[string]Group{
					"all": {
						Hosts: map[string]variables.Variables{
							"foo": {},
							"bar": {},
							"baz": {},
						},
					},
				},
			},
			hostName: "bar",
			want:     map[string]struct{}{"all": {}},
		},
		{
			name: "in multiple sets",
			inventory: Inventory{
				Groups: map[string]Group{
					"myGroup": {
						Hosts: map[string]variables.Variables{
							"foo": {},
						},
						Children: map[string]Group{
							"fruit": {
								Hosts: map[string]variables.Variables{
									"myBanana": {},
								},
								Children: map[string]Group{
									"apple": {
										Hosts: map[string]variables.Variables{
											"myGala": {},
										},
									},
								},
							},
							"party": {
								Hosts: map[string]variables.Variables{
									"myGala": {},
								},
							},
						},
					},
				},
			},
			hostName: "myGala",
			want:     map[string]struct{}{"all": {}, "myGroup": {}, "fruit": {}, "apple": {}, "party": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.inventory.Find(tt.hostName)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupAll(t *testing.T) {
	tests := []struct {
		name  string
		group Group
		want  map[string]struct{}
	}{
		{
			name: "simple",
			group: Group{
				Hosts: map[string]variables.Variables{
					"foo": {},
					"bar": {},
					"baz": {},
				},
			},
			want: map[string]struct{}{
				"foo": {},
				"bar": {},
				"baz": {},
			},
		},
		{
			name: "nested",
			group: Group{
				Hosts: map[string]variables.Variables{
					"foo": {},
				},
				Children: map[string]Group{
					"fruit": {
						Hosts: map[string]variables.Variables{
							"myBanana": {},
						},
						Children: map[string]Group{
							"apple": {
								Hosts: map[string]variables.Variables{
									"myGala": {},
								},
							},
						},
					},
					"party": {
						Hosts: map[string]variables.Variables{
							"myGala": {},
						},
					},
				},
			},
			want: map[string]struct{}{
				"foo":      {},
				"myBanana": {},
				"myGala":   {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.group.All()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInventoryAll(t *testing.T) {
	tests := []struct {
		name      string
		inventory Inventory
		want      map[string]struct{}
	}{
		{
			name: "nested groups",
			inventory: Inventory{
				Groups: map[string]Group{
					"myGroup": {
						Hosts: map[string]variables.Variables{
							"foo": {},
						},
						Children: map[string]Group{
							"fruit": {
								Hosts: map[string]variables.Variables{
									"myBanana": {},
								},
								Children: map[string]Group{
									"apple": {
										Hosts: map[string]variables.Variables{
											"myGala": {},
										},
									},
								},
							},
							"party": {
								Hosts: map[string]variables.Variables{
									"myGala": {},
								},
							},
						},
					},
				},
			},
			want: map[string]struct{}{
				"foo":      {},
				"myBanana": {},
				"myGala":   {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.inventory.All()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupNodeVars(t *testing.T) {
	tests := []struct {
		name          string
		group         Group
		hostName      string
		wantHostVars  variables.Variables
		wantGroupVars variables.Variables
	}{
		{
			name: "simple",
			group: Group{
				Hosts: map[string]variables.Variables{
					"foo": {
						"hello":  "world!",
						"answer": 42,
					},
				},
				Vars: variables.Variables{
					"ansible_port": 2222,
				},
			},
			hostName: "foo",
			wantHostVars: variables.Variables{
				"hello":  "world!",
				"answer": 42,
			},
			wantGroupVars: variables.Variables{
				"ansible_port": 2222,
			},
		},
		{
			name: "nested",
			group: Group{
				Hosts: map[string]variables.Variables{
					"foo": {},
				},
				Vars: variables.Variables{
					"hello": "world!",
				},
				Children: map[string]Group{
					"someChild": {
						Hosts: map[string]variables.Variables{
							"bar": {
								"fruit": "banana",
							},
						},
						Vars: variables.Variables{
							"answer": 42,
						},
						Children: map[string]Group{
							"nestedChild": {
								Vars: variables.Variables{
									"pineapple": "notonpizza",
								},
							},
						},
					},
				},
			},
			hostName: "bar",
			wantHostVars: variables.Variables{
				"fruit": "banana",
			},
			wantGroupVars: variables.Variables{
				"hello":  "world!",
				"answer": 42,
			},
		},
		{
			name: "with override",
			group: Group{
				Vars: variables.Variables{
					"answer": 43,
					"hello":  "country!",
				},
				Children: map[string]Group{
					"moreCorrect": {
						Hosts: map[string]variables.Variables{
							"foo": {
								"hello": "world!",
							},
						},
						Vars: variables.Variables{
							"answer": 42,
						},
					},
					"ignored": {
						Vars: variables.Variables{
							"hello": "someone!",
						},
					},
				},
			},
			hostName: "foo",
			wantHostVars: variables.Variables{
				"hello": "world!",
			},
			wantGroupVars: variables.Variables{
				"answer": 42,
				"hello":  "country!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHostVars, gotGroupVars := tt.group.NodeVars(tt.hostName)
			if diff := cmp.Diff(tt.wantHostVars, gotHostVars); diff != "" {
				t.Errorf("host vars mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantGroupVars, gotGroupVars); diff != "" {
				t.Errorf("group vars mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInventoryNodeVars(t *testing.T) {
	tests := []struct {
		name      string
		inventory Inventory
		hostName  string
		want      variables.Variables
	}{
		{
			name: "merged vars with override",
			inventory: Inventory{
				Groups: map[string]Group{
					"top": {
						Vars: variables.Variables{
							"answer": 43,
							"hello":  "country!",
						},
						Children: map[string]Group{
							"moreCorrect": {
								Hosts: map[string]variables.Variables{
									"foo": {
										"hello":     "world!",
										"pineapple": "not on pizza",
									},
								},
								Vars: variables.Variables{
									"answer":    42,
									"pineapple": "on pizza",
								},
							},
							"ignored": {
								Vars: variables.Variables{
									"hello": "someone!",
								},
							},
						},
					},
				},
			},
			hostName: "foo",
			want: variables.Variables{
				"hello":     "world!",
				"answer":    42,
				"pineapple": "not on pizza",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.inventory.NodeVars(tt.hostName)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
