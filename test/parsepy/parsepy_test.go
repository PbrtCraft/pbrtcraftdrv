package parsepy_test

import (
	"testing"

	"github.com/PbrtCraft/pbrtcraftdrv/parsepy"
	"github.com/google/go-cmp/cmp"
)

// TestParse test parsing
func TestParse(t *testing.T) {
	got, err := parsepy.GetClasses("test.py")
	if err != nil {
		t.Fail()
	}

	want := []*parsepy.Class{
		{
			Def:  "class TestA:",
			Name: "TestA",
			Doc:  "A is a\nJzzzz\n",
			InitFunc: parsepy.Function{
				Def: "def __init__(self, a: int, b: str, c: float = 1):",
				Doc: "Initial\n:param a: apple\n:param b: bus\n",
				Params: []*parsepy.Param{
					{Name: "a", Doc: "apple", Type: "int", DefaultValue: "0"},
					{Name: "b", Doc: "bus", Type: "str"},
					{Name: "c", Type: "float", DefaultValue: "1"},
				},
			},
		},
		{
			Def:  "class TestB:",
			Name: "TestB",
			Doc:  "B is b",
			InitFunc: parsepy.Function{
				Def: "def __init__(self, a, c: float):",
				Doc: "Initial\n- a apple\n- b bus\n",
				Params: []*parsepy.Param{
					{Name: "a", Type: "int", DefaultValue: "0"},
					{Name: "c", Type: "float", DefaultValue: "0.0"},
				},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
