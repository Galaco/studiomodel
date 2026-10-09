package phy

import (
	"reflect"
	"slices"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

const ragdollText = `solid {
"index" "0"
"name" "pelvis"
"mass" "13.5"
"surfaceprop" "flesh"
"damping" "0.000000"
"rotdamping" "0.000000"
"inertia" "10.000000"
"volume" "4224.656250"
}
solid {
"index" "1"
"name" "spine"
"parent" "pelvis"
"mass" "2"
"surfaceprop" "flesh"
}
ragdollconstraint {
"parent" "0"
"child" "1"
"xmin" "-30.000000"
"xmax" "30.000000"
"xfriction" "0.5"
"ymin" "-10"
"ymax" "10"
"yfriction" "0"
"zmin" "-45"
"zmax" "20"
"zfriction" "0"
}
collisionrules {
"collisionpair" "0,1"
"collisionpair" "1,2"
}
editparams {
"rootname" ""
"totalmass" "15.5"
}
`

func TestSections(t *testing.T) {
	sections, err := (&Phy{Text: ragdollText}).Sections()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, section := range sections {
		names = append(names, section.Name)
	}
	if want := []string{"solid", "solid", "ragdollconstraint", "collisionrules", "editparams"}; !slices.Equal(names, want) {
		t.Errorf("got sections %q, want %q", names, want)
	}
	if value, ok := sections[4].Value("rootname"); !ok || value != "" {
		t.Errorf("editparams rootname: got %q, %v", value, ok)
	}
	if _, ok := sections[4].Value("concave"); ok {
		t.Error("editparams concave: expected no value")
	}
}

func TestSolidInfos(t *testing.T) {
	solids, err := (&Phy{Text: ragdollText}).SolidInfos()
	if err != nil {
		t.Fatal(err)
	}
	want := []SolidInfo{
		{Index: 0, Name: "pelvis", Mass: 13.5, SurfaceProp: "flesh", Inertia: 10, Volume: 4224.65625},
		{Index: 1, Name: "spine", Parent: "pelvis", Mass: 2, SurfaceProp: "flesh"},
	}
	if !reflect.DeepEqual(solids, want) {
		t.Errorf("got %+v, want %+v", solids, want)
	}
}

func TestRagdollConstraints(t *testing.T) {
	constraints, err := (&Phy{Text: ragdollText}).RagdollConstraints()
	if err != nil {
		t.Fatal(err)
	}
	want := []RagdollConstraint{{Parent: 0, Child: 1, XMin: -30, XMax: 30, XFriction: 0.5, YMin: -10, YMax: 10, ZMin: -45, ZMax: 20}}
	if !reflect.DeepEqual(constraints, want) {
		t.Errorf("got %+v, want %+v", constraints, want)
	}
}

func TestCollisionPairs(t *testing.T) {
	pairs, err := (&Phy{Text: ragdollText}).CollisionPairs()
	if err != nil {
		t.Fatal(err)
	}
	if want := [][2]int{{0, 1}, {1, 2}}; !reflect.DeepEqual(pairs, want) {
		t.Errorf("got %v, want %v", pairs, want)
	}
}

func TestKeyValuesRejectMalformedText(t *testing.T) {
	for name, text := range map[string]string{
		"unclosed section":      "solid {\n\"index\" \"0\"\n",
		"unterminated string":   "solid {\n\"index\" \"0\n}",
		"key without value":     "solid {\n\"index\"\n}",
		"bad number":            "solid {\n\"index\" \"zero\"\n}",
		"bad collision pair":    "collisionrules {\n\"collisionpair\" \"0\"\n}",
		"bad constraint number": "ragdollconstraint {\n\"xmin\" \"-\"\n}",
	} {
		t.Run(name, func(t *testing.T) {
			p := &Phy{Text: text}
			_, err1 := p.SolidInfos()
			_, err2 := p.CollisionPairs()
			_, err3 := p.RagdollConstraints()
			if err1 == nil && err2 == nil && err3 == nil {
				t.Error("expected an error")
			}
		})
	}
}

// Like the engine, words that don't start a section are skipped, as in some of Valve's own files
func TestSectionsSkipStrayWords(t *testing.T) {
	text := `solid {
"index" "0"
}
break {"model" "chunk01" "fadetime" "20"  }
break break {"model" "chunk09" "fadetime" "20"  }
break {"model" "chunk10" "break" {"model" "chunk11" }
editparams {
"totalmass" "25"
}
`
	sections, err := (&Phy{Text: text}).Sections()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, section := range sections {
		names = append(names, section.Name)
	}
	if want := []string{"solid", "break", "break", "editparams"}; !slices.Equal(names, want) {
		t.Errorf("got sections %q, want %q", names, want)
	}
	// An unclosed section takes in the one after it, including its {
	if value, _ := sections[2].Value("break"); value != "{" {
		t.Errorf("unclosed break section: got %+v", sections[2])
	}
}

func TestToSource(t *testing.T) {
	// IVP x, y (down), z in metres -> Source x, y, z (up) in inches
	got := ToSource(mgl32.Vec4{0.0254, 0.0508, 0.0762, 0})
	if want := (mgl32.Vec3{1, 3, -2}); !got.ApproxEqual(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
