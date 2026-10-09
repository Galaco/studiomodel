package phy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-gl/mathgl/mgl32"
)

// Section is one block of a phy's key-values text, such as a solid or a ragdoll constraint
type Section struct {
	// Name - solid, ragdollconstraint, collisionrules, editparams or animatedfriction
	Name string
	// KeyValues in file order. Keys can repeat, such as collisionpair.
	KeyValues []KeyValue
}

// KeyValue
type KeyValue struct {
	Key   string
	Value string
}

// Value returns the first value for key
func (section *Section) Value(key string) (string, bool) {
	for _, kv := range section.KeyValues {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

// SolidInfo describes one of a phy's solids
type SolidInfo struct {
	// Index of the solid in Phy.Solids
	Index int
	// Name of the bone the solid follows
	Name string
	// Parent is the name of the parent bone, empty for the root solid
	Parent string
	// Mass in kilograms
	Mass float32
	// SurfaceProp
	SurfaceProp string
	// Damping
	Damping float32
	// RotDamping
	RotDamping float32
	// Inertia
	Inertia float32
	// Volume
	Volume float32
}

// RagdollConstraint limits how a child solid rotates relative to its parent. Limits are in degrees.
type RagdollConstraint struct {
	// Parent - index of the parent solid
	Parent int
	// Child - index of the child solid
	Child int

	XMin, XMax, XFriction float32
	YMin, YMax, YFriction float32
	ZMin, ZMax, ZFriction float32
}

// metresPerInch converts between IVP's metres and Source's inches
const metresPerInch = 0.0254

// ToSource converts a vertex from IVP space, which is in metres with y pointing down, to Source space, which is in
// inches with z pointing up
func ToSource(v mgl32.Vec4) mgl32.Vec3 {
	return mgl32.Vec3{v[0], v[2], -v[1]}.Mul(1 / metresPerInch)
}

// Sections parses Text into its sections
func (phy *Phy) Sections() ([]Section, error) {
	tokens, err := tokenize(phy.Text)
	if err != nil {
		return nil, err
	}

	var sections []Section
	for i := 0; i < len(tokens); {
		if tokens[i].brace != 0 || i+1 >= len(tokens) || tokens[i+1].brace != '{' {
			return nil, fmt.Errorf("expected a section name and {, got %q", tokens[i].text)
		}
		section := Section{Name: tokens[i].text}
		for i += 2; ; i += 2 {
			if i >= len(tokens) {
				return nil, fmt.Errorf("section %q is not closed", section.Name)
			}
			if tokens[i].brace == '}' {
				i++
				break
			}
			if tokens[i].brace != 0 || i+1 >= len(tokens) || tokens[i+1].brace != 0 {
				return nil, fmt.Errorf("section %q: expected a key and value, got %q", section.Name, tokens[i].text)
			}
			section.KeyValues = append(section.KeyValues, KeyValue{Key: tokens[i].text, Value: tokens[i+1].text})
		}
		sections = append(sections, section)
	}
	return sections, nil
}

// SolidInfos returns the description of each solid in the key-values text
func (phy *Phy) SolidInfos() ([]SolidInfo, error) {
	sections, err := phy.Sections()
	if err != nil {
		return nil, err
	}

	var out []SolidInfo
	for _, section := range sections {
		if section.Name != "solid" {
			continue
		}
		p := sectionParser{section: &section}
		solid := SolidInfo{
			Index:       p.int("index"),
			Name:        p.string("name"),
			Parent:      p.string("parent"),
			Mass:        p.float("mass"),
			SurfaceProp: p.string("surfaceprop"),
			Damping:     p.float("damping"),
			RotDamping:  p.float("rotdamping"),
			Inertia:     p.float("inertia"),
			Volume:      p.float("volume"),
		}
		if p.err != nil {
			return nil, p.err
		}
		out = append(out, solid)
	}
	return out, nil
}

// RagdollConstraints returns the ragdoll constraints in the key-values text
func (phy *Phy) RagdollConstraints() ([]RagdollConstraint, error) {
	sections, err := phy.Sections()
	if err != nil {
		return nil, err
	}

	var out []RagdollConstraint
	for _, section := range sections {
		if section.Name != "ragdollconstraint" {
			continue
		}
		p := sectionParser{section: &section}
		constraint := RagdollConstraint{
			Parent:    p.int("parent"),
			Child:     p.int("child"),
			XMin:      p.float("xmin"),
			XMax:      p.float("xmax"),
			XFriction: p.float("xfriction"),
			YMin:      p.float("ymin"),
			YMax:      p.float("ymax"),
			YFriction: p.float("yfriction"),
			ZMin:      p.float("zmin"),
			ZMax:      p.float("zmax"),
			ZFriction: p.float("zfriction"),
		}
		if p.err != nil {
			return nil, p.err
		}
		out = append(out, constraint)
	}
	return out, nil
}

// CollisionPairs returns the pairs of solids that collide with each other, from the collision rules
func (phy *Phy) CollisionPairs() ([][2]int, error) {
	sections, err := phy.Sections()
	if err != nil {
		return nil, err
	}

	var out [][2]int
	for _, section := range sections {
		if section.Name != "collisionrules" {
			continue
		}
		for _, kv := range section.KeyValues {
			if kv.Key != "collisionpair" {
				continue
			}
			a, b, ok := strings.Cut(kv.Value, ",")
			first, errA := strconv.Atoi(strings.TrimSpace(a))
			second, errB := strconv.Atoi(strings.TrimSpace(b))
			if !ok || errA != nil || errB != nil {
				return nil, fmt.Errorf("invalid collision pair %q", kv.Value)
			}
			out = append(out, [2]int{first, second})
		}
	}
	return out, nil
}

// sectionParser reads typed values from a section, keeping the first error. Missing keys read as zero.
type sectionParser struct {
	section *Section
	err     error
}

func (p *sectionParser) string(key string) string {
	value, _ := p.section.Value(key)
	return value
}

func (p *sectionParser) int(key string) int {
	value, ok := p.section.Value(key)
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil && p.err == nil {
		p.err = fmt.Errorf("section %q: %s: %w", p.section.Name, key, err)
	}
	return n
}

func (p *sectionParser) float(key string) float32 {
	value, ok := p.section.Value(key)
	if !ok {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(value), 32)
	if err != nil && p.err == nil {
		p.err = fmt.Errorf("section %q: %s: %w", p.section.Name, key, err)
	}
	return float32(f)
}

// token is a word, a quoted string, or a brace
type token struct {
	text  string
	brace byte
}

// tokenize splits key-values text into words, quoted strings and braces
func tokenize(text string) ([]token, error) {
	var tokens []token
	for i := 0; i < len(text); {
		switch c := text[i]; {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '{' || c == '}':
			tokens = append(tokens, token{text: string(c), brace: c})
			i++
		case c == '"':
			end := strings.IndexByte(text[i+1:], '"')
			if end < 0 {
				return nil, fmt.Errorf("unterminated string at offset %d", i)
			}
			tokens = append(tokens, token{text: text[i+1 : i+1+end]})
			i += end + 2
		default:
			end := i
			for end < len(text) && !strings.ContainsRune(" \t\r\n{}\"", rune(text[end])) {
				end++
			}
			tokens = append(tokens, token{text: text[i:end]})
			i = end
		}
	}
	return tokens, nil
}
