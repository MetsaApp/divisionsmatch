package divisionsmatch_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	dm "github.com/MetsaApp/divisionsmatch"
)

// Every embedded rulebook parses and passes its checks: a broken file
// cannot ship, which is what lets Latest panic on one.
func TestEmbeddedRulebooksParse(t *testing.T) {
	versions := dm.Rulebooks()
	must(t, len(versions) > 0, "no embedded rulebook")
	for _, v := range versions {
		rb, err := dm.Embedded(v)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		equal(t, v, rb.Version, "the file name is the version")
	}
	equal(t, versions[len(versions)-1], dm.Latest().Version)
	_, err := dm.Embedded("1999")
	must(t, err != nil, "an unknown version is an error")
}

// The rulebook is the 2025 Danish divisionsmatch rules: a group per
// rulebook class, each with the points its scoring runners take in
// finishing order. Half the scale is the most one club can take.
func TestRulebookPoints(t *testing.T) {
	rb := rulebook(t)
	cases := []struct {
		group  string
		points []int
	}{
		{"H21", []int{8, 7, 6, 5, 4, 3, 2, 1}},
		{"D21", []int{8, 7, 6, 5, 4, 3, 2, 1}},
		{"D50", []int{6, 5, 4, 3, 2, 1}},
		{"H40", []int{6, 5, 4, 3, 2, 1}},
		{"H16", []int{4, 3, 2, 1}},
		{"D80", []int{4, 3, 2, 1}},
		{"H14B", []int{2, 2, 1, 1}},
		{"D21B", []int{2, 2, 2, 1, 1, 1}},
		{"H-Let", []int{2, 2, 2, 1, 1, 1}},
		{"Begynder", []int{1, 1, 1, 1, 1, 1}},
		{"D10", []int{1, 1, 1, 1, 1, 1}},
	}
	for _, c := range cases {
		g, found := rb.Group(c.group)
		must(t, found, c.group)
		equal(t, c.points, g.Scale, c.group)
	}
	_, found := rb.Group("H21A")
	equal(t, false, found, "not a rulebook group")
	equal(t, dm.MatchPoints{Win: 2, Draw: 1, Loss: 0}, rb.MatchPoints)
	equal(t, 0.5, rb.ClubLimit)
	equal(t, 0.5, rb.Walkover.ScoreShare)
	equal(t, []dm.TieBreak{dm.TieBreakMutualScore, dm.TieBreakBestRunners}, rb.TieBreaks)
}

// The rulebook lists its groups in one order, the women's then the men's
// with the beginners first, which is the order a printout shows them in.
func TestRulebookOrder(t *testing.T) {
	groups := rulebook(t).Groups
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	equal(t, "Begynder", names[0])
	equal(t, "D10", names[1])
	equal(t, "H-Let", names[len(names)-1])
	equal(t, 37, len(names))
	equal(t, "D-Let", names[18], "the women's groups end before the men's begin")
	equal(t, "H10", names[19])
}

// A competition's class names are the timing software's, not the
// rulebook's: "H21-", "H-20", "D 50", "H21-B". The guess pairs the ones
// that are the same class under a different spelling and leaves the rest
// to the organiser.
func TestGuessGroup(t *testing.T) {
	cases := map[string]string{
		"H21":      "H21",
		"H21-":     "H21",
		"H-20":     "H20",
		"D 50":     "D50",
		"H21-B":    "H21B",
		"H-20B":    "H20B",
		"d21":      "D21",
		"Begynder": "Begynder",
		"Beg":      "Begynder",
		"H-Let":    "H-Let",
		"H let":    "H-Let",
		"D-let":    "D-Let",
		"H21A":     "H21",
		"H21-A":    "H21",
		"H-20C":    "",
		"Åben 1":   "",
		"Open":     "",
		"":         "",
	}
	rb := rulebook(t)
	for name, want := range cases {
		equal(t, want, rb.GuessGroup(name), name)
	}
}

// A config is put in order: a division with nothing in it is dropped, a
// club is listed once and only its own clubs can be absent, and a group
// the rulebook lacks is no group. The config given is left as it was.
func TestNormalise(t *testing.T) {
	in := dm.Config{
		Divisions: []dm.Division{
			{Name: "  2. division ", Clubs: []string{"E", " G", "E", "", "I"}, Absent: []string{"I", "K"}},
			{Name: " ", Clubs: nil},
			{Name: "", Clubs: []string{"C"}},
		},
		Groups: map[string]string{"H21-": "H21", "Åben": "Åben", "H-20": ""},
	}
	cfg := dm.Normalise(rulebook(t), in)
	equal(t, []dm.Division{
		{Name: "2. division", Clubs: []string{"E", "G", "I"}, Absent: []string{"I"}},
		{Name: "", Clubs: []string{"C"}, Absent: []string{}},
	}, cfg.Divisions)
	equal(t, map[string]string{"H21-": "H21", "Åben": "", "H-20": ""}, cfg.Groups)
	equal(t, "Åben", in.Groups["Åben"], "the input is not changed")
	equal(t, "  2. division ", in.Divisions[0].Name)

	empty := dm.Normalise(rulebook(t), dm.Config{})
	must(t, empty.Divisions != nil && empty.Groups != nil, "an empty config normalises to empty lists")
}

// A rulebook file is checked when it is read, so someone who breaks one
// learns why from the error, not from a wrong standing. Each case is
// the embedded 2025 file with one change.
func TestParseRefusesBrokenRulebooks(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("rulebooks", "2025.yml"))
	if err != nil {
		t.Fatal(err)
	}
	base := string(raw)
	cases := map[string][2]string{
		"unknown key":         {"club_limit: 0.5", "club_limt: 0.5"},
		"club limit above 1":  {"club_limit: 0.5", "club_limit: 1.5"},
		"draw above win":      {"draw: 1", "draw: 3"},
		"unknown tie-break":   {"  - best_runners", "  - coin_toss"},
		"repeated tie-break":  {"  - best_runners", "  - mutual_score"},
		"zero in a scale":     {"scale: [4, 3, 2, 1]", "scale: [4, 3, 0, 1]"},
		"empty scale":         {"scale: [8, 7, 6, 5, 4, 3, 2, 1]\n  - name: D21B", "scale: []\n  - name: D21B"},
		"alias of two groups": {"aliases: [Beg]", "aliases: [D-10]"},
		"walkover share":      {"score_share: 0.5", "score_share: -1"},
		"no version":          {`version: "2025"`, `version: ""`},
	}
	for name, c := range cases {
		must(t, strings.Contains(base, c[0]), name, ": the case does not apply to the file")
		if _, parseErr := dm.Parse([]byte(strings.Replace(base, c[0], c[1], 1))); parseErr == nil {
			t.Errorf("%s: parsed without error", name)
		}
	}
	if _, err = dm.Parse(raw); err != nil {
		t.Fatalf("the file itself: %v", err)
	}
}

// A rulebook can be read from disk, for a new rulebook version being tried
// before it is merged.
func TestLoad(t *testing.T) {
	rb, err := dm.Load(filepath.Join("rulebooks", "2025.yml"))
	if err != nil {
		t.Fatal(err)
	}
	equal(t, "2025", rb.Version)
	_, err = dm.Load(filepath.Join(t.TempDir(), "missing.yml"))
	must(t, err != nil, "a missing file is an error")
}

// The JSON Schema that editors validate a rulebook with names the same
// keys the Go reader knows, so the two cannot drift apart unseen.
func TestSchemaMatchesRulebook(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("schema", "rulebook.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err = json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	var keys []string
	rt := reflect.TypeFor[dm.Rulebook]()
	for f := range rt.Fields() {
		keys = append(keys, f.Tag.Get("yaml"))
	}
	var props []string
	for k := range schema.Properties {
		props = append(props, k)
	}
	slices.Sort(keys)
	slices.Sort(props)
	equal(t, keys, props)
}
