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
			"usa": Group{
				Children: map[string]Group{
					"northeast": Group{},
					"northwest": Group{},
					"southwest": Group{},
					"southeast": Group{
						Vars: map[string]any{
							"some_server":             "foo.southeast.example.com",
							"halon_system_timeout":    uint64(30),
							"self_destruct_countdown": uint64(60),
							"escape_pods":             uint64(2),
						},
						Children: map[string]Group{
							"atlanta": Group{
								Hosts: map[string]variables.Variables{
									"host1": nil,
									"host2": nil,
								},
							},
							"raleigh": Group{
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
			"foo": variables.Variables{},
			"bar": variables.Variables{},
			"baz": variables.Variables{},
		},
	}

	got := a.Find("top", "bar")
	expected := map[string]struct{}{"top": struct{}{}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupFindNested(t *testing.T) {
	a := Group{
		Hosts: map[string]variables.Variables{
			"foo": variables.Variables{},
		},
		Children: map[string]Group{
			"fruit": Group{
				Hosts: map[string]variables.Variables{
					"myBanana": variables.Variables{},
				},
				Children: map[string]Group{
					"apple": Group{
						Hosts: map[string]variables.Variables{
							"myGala": variables.Variables{},
						},
					},
				},
			},
		},
	}

	got := a.Find("top", "myBanana")
	expected := map[string]struct{}{"fruit": struct{}{}, "top": struct{}{}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}

	got = a.Find("top", "myGala")
	expected = map[string]struct{}{"apple": struct{}{}, "fruit": struct{}{}, "top": struct{}{}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupFindInMultipleSets(t *testing.T) {
	a := Group{
		Hosts: map[string]variables.Variables{
			"foo": variables.Variables{},
		},
		Children: map[string]Group{
			"fruit": Group{
				Hosts: map[string]variables.Variables{
					"myBanana": variables.Variables{},
				},
				Children: map[string]Group{
					"apple": Group{
						Hosts: map[string]variables.Variables{
							"myGala": variables.Variables{},
						},
					},
				},
			},
			"party": Group{
				Hosts: map[string]variables.Variables{
					"myGala": variables.Variables{},
				},
			},
		},
	}

	got := a.Find("top", "myGala")
	expected := map[string]struct{}{"apple": struct{}{}, "fruit": struct{}{}, "party": struct{}{}, "top": struct{}{}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestInventoryFindSimple(t *testing.T) {
	a := Inventory{
		Groups: map[string]Group{
			"all": Group{
				Hosts: map[string]variables.Variables{
					"foo": variables.Variables{},
					"bar": variables.Variables{},
					"baz": variables.Variables{},
				},
			},
		},
	}

	got := a.Find("bar")
	expected := map[string]struct{}{"all": struct{}{}}
	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestInventoryFindInMultipleSets(t *testing.T) {
	a := Inventory{
		Groups: map[string]Group{
			"myGroup": Group{
				Hosts: map[string]variables.Variables{
					"foo": variables.Variables{},
				},
				Children: map[string]Group{
					"fruit": Group{
						Hosts: map[string]variables.Variables{
							"myBanana": variables.Variables{},
						},
						Children: map[string]Group{
							"apple": Group{
								Hosts: map[string]variables.Variables{
									"myGala": variables.Variables{},
								},
							},
						},
					},
					"party": Group{
						Hosts: map[string]variables.Variables{
							"myGala": variables.Variables{},
						},
					},
				},
			},
		},
	}

	got := a.Find("myGala")
	expected := map[string]struct{}{"all": struct{}{}, "myGroup": struct{}{}, "fruit": struct{}{}, "apple": struct{}{}, "party": struct{}{}}

	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupAllSimple(t *testing.T) {
	a := Group{
		Hosts: map[string]variables.Variables{
			"foo": variables.Variables{},
			"bar": variables.Variables{},
			"baz": variables.Variables{},
		},
	}

	got := a.All()
	expected := map[string]struct{}{
		"foo": struct{}{},
		"bar": struct{}{},
		"baz": struct{}{},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupAllNested(t *testing.T) {
	a := Group{
		Hosts: map[string]variables.Variables{
			"foo": variables.Variables{},
		},
		Children: map[string]Group{
			"fruit": Group{
				Hosts: map[string]variables.Variables{
					"myBanana": variables.Variables{},
				},
				Children: map[string]Group{
					"apple": Group{
						Hosts: map[string]variables.Variables{
							"myGala": variables.Variables{},
						},
					},
				},
			},
			"party": Group{
				Hosts: map[string]variables.Variables{
					"myGala": variables.Variables{},
				},
			},
		},
	}

	got := a.All()
	expected := map[string]struct{}{
		"foo":      struct{}{},
		"myBanana": struct{}{},
		"myGala":   struct{}{},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestGroupNodeVars(t *testing.T) {
	g := Group{
		Hosts: map[string]variables.Variables{
			"foo": variables.Variables{
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
			"foo": variables.Variables{},
		},
		Vars: variables.Variables{
			"hello": "world!",
		},
		Children: map[string]Group{
			"someChild": Group{
				Hosts: map[string]variables.Variables{
					"bar": variables.Variables{
						"fruit": "banana",
					},
				},
				Vars: variables.Variables{
					"answer": 42,
				},
				Children: map[string]Group{
					"nestedChild": Group{
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
			"moreCorrect": Group{
				Hosts: map[string]variables.Variables{
					"foo": variables.Variables{
						"hello": "world!",
					},
				},
				Vars: variables.Variables{
					"answer": 42,
				},
			},
			"ignored": Group{
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
			"top": Group{
				Vars: variables.Variables{
					"answer": 43,
					"hello":  "country!",
				},
				Children: map[string]Group{
					"moreCorrect": Group{
						Hosts: map[string]variables.Variables{
							"foo": variables.Variables{
								"hello":     "world!",
								"pineapple": "not on pizza",
							},
						},
						Vars: variables.Variables{
							"answer":    42,
							"pineapple": "on pizza",
						},
					},
					"ignored": Group{
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
