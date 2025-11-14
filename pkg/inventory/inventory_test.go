package inventory

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestUnmarshalYAML(t *testing.T) {
	expected := Inventory{
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
	}

	y := []byte(`
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
`)

	var got Inventory
	if err := yaml.Unmarshal(y, &got); err != nil {
		t.Error(err)
	}

	if !cmp.Equal(got, expected, cmpopts.EquateEmpty()) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestGroupFindSimple(t *testing.T) {
	a := Group{
		Hosts: map[string]variables.Variables{
			"foo": {},
			"bar": {},
			"baz": {},
		},
	}

	got := a.Find("top", "bar")
	expected := map[string]struct{}{"top": {}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupFindNested(t *testing.T) {
	a := Group{
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
	}

	got := a.Find("top", "myBanana")
	expected := map[string]struct{}{"fruit": {}, "top": {}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}

	got = a.Find("top", "myGala")
	expected = map[string]struct{}{"apple": {}, "fruit": {}, "top": {}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupFindInMultipleSets(t *testing.T) {
	a := Group{
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
	}

	got := a.Find("top", "myGala")
	expected := map[string]struct{}{"apple": {}, "fruit": {}, "party": {}, "top": {}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestInventoryFindSimple(t *testing.T) {
	a := Inventory{
		Groups: map[string]Group{
			"all": {
				Hosts: map[string]variables.Variables{
					"foo": {},
					"bar": {},
					"baz": {},
				},
			},
		},
	}

	got := a.Find("bar")
	expected := map[string]struct{}{"all": {}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestInventoryFindInMultipleSets(t *testing.T) {
	a := Inventory{
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
	}

	got := a.Find("myGala")
	expected := map[string]struct{}{"all": {}, "myGroup": {}, "fruit": {}, "apple": {}, "party": {}}

	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupAllSimple(t *testing.T) {
	a := Group{
		Hosts: map[string]variables.Variables{
			"foo": {},
			"bar": {},
			"baz": {},
		},
	}

	got := a.All()
	expected := map[string]struct{}{
		"foo": {},
		"bar": {},
		"baz": {},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupAllNested(t *testing.T) {
	a := Group{
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
	}

	got := a.All()
	expected := map[string]struct{}{
		"foo":      {},
		"myBanana": {},
		"myGala":   {},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestInventoryAll(t *testing.T) {
	i := Inventory{
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
	}

	got := i.All()
	expected := map[string]struct{}{
		"foo":      {},
		"myBanana": {},
		"myGala":   {},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupNodeVars(t *testing.T) {
	g := Group{
		Hosts: map[string]variables.Variables{
			"foo": {
				"hello":  "world!",
				"answer": 42,
			},
		},
		Vars: variables.Variables{
			"ansible_port": 2222,
		},
	}

	hostVars, groupVars := g.NodeVars("foo")
	expectedHostVars := variables.Variables{
		"hello":  "world!",
		"answer": 42,
	}
	expectedGroupVars := variables.Variables{
		"ansible_port": 2222,
	}

	if !cmp.Equal(hostVars, expectedHostVars) {
		t.Errorf("expected %#v but got %#v", expectedHostVars, hostVars)
	}

	if !cmp.Equal(groupVars, expectedGroupVars) {
		t.Errorf("expected %#v but got %#v", expectedGroupVars, groupVars)
	}
}

func TestGroupNodeVarsNested(t *testing.T) {
	g := Group{
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
	}

	hostVars, groupVars := g.NodeVars("bar")
	expectedHostVars := variables.Variables{
		"fruit": "banana",
	}
	expectedGroupVars := variables.Variables{
		"hello":  "world!",
		"answer": 42,
	}

	if !cmp.Equal(hostVars, expectedHostVars) {
		t.Errorf("expected %#v but got %#v", expectedHostVars, hostVars)
	}

	if !cmp.Equal(groupVars, expectedGroupVars) {
		t.Errorf("expected %#v but got %#v", expectedGroupVars, groupVars)
	}
}

func TestGroupNodeVarsOverride(t *testing.T) {
	g := Group{
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
	}

	hostVars, groupVars := g.NodeVars("foo")
	expectedHostVars := variables.Variables{
		"hello": "world!",
	}
	expectedGroupVars := variables.Variables{
		"answer": 42,
		"hello":  "country!",
	}

	if !cmp.Equal(hostVars, expectedHostVars) {
		t.Errorf("expected %#v but got %#v", expectedHostVars, hostVars)
	}

	if !cmp.Equal(groupVars, expectedGroupVars) {
		t.Errorf("expected %#v but got %#v", expectedGroupVars, groupVars)
	}
}

func TestInventoryNodeVars(t *testing.T) {
	i := Inventory{
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
	}

	got := i.NodeVars("foo")
	expected := variables.Variables{
		"hello":     "world!",
		"answer":    42,
		"pineapple": "not on pizza",
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}
