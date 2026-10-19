package divisionsmatch

import (
	"bytes"
	"cmp"
	"embed"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// The rulebooks ship inside the module, one YAML file per version, so a
// program that imports it needs no files on disk.
//
//go:embed rulebooks/*.yml
var rulebookFiles embed.FS

// Rulebook is one version of the rules, read from YAML. Everything in it
// that a federation can change by vote is data; the steps that apply it
// (the finish order, shared points, the tie-break mechanics) are Go.
type Rulebook struct {
	// Version names the rulebook: "2025".
	Version string `yaml:"version"`
	// Name and Source are for people: what the rules are and where they
	// come from.
	Name   string `yaml:"name"`
	Source string `yaml:"source"`

	MatchPoints MatchPoints `yaml:"match_points"`
	// ClubLimit is the share of a group's scale one club can score with:
	// 0.5 is four runners in the eight places of H21.
	ClubLimit  float64         `yaml:"club_limit"`
	Walkover   Walkover        `yaml:"walkover"`
	TieBreaks  []TieBreak      `yaml:"tie_breaks"`
	ClassNames ClassNames      `yaml:"class_names"`
	Groups     []RulebookGroup `yaml:"groups"`
}

// MatchPoints is what one match gives each club.
type MatchPoints struct {
	Win  int `yaml:"win"`
	Draw int `yaml:"draw"`
	Loss int `yaml:"loss"`
}

// Walkover is the rule for a match against a club that did not turn up:
// the club that came wins on this share of its own score.
type Walkover struct {
	ScoreShare float64 `yaml:"score_share"`
}

// ClassNames is how a competition's class names are read into groups.
// Endings in SameGroupSuffixes are dropped when the full name is no
// group: an A class is the group itself.
type ClassNames struct {
	SameGroupSuffixes []string `yaml:"same_group_suffixes"`
}

// TieBreak names one way to separate clubs level on match points.
type TieBreak string

// The tie-breaks a rulebook can list, in any order.
const (
	// TieBreakMutualScore orders level clubs by the score each took in
	// the matches between them.
	TieBreakMutualScore TieBreak = "mutual_score"
	// TieBreakBestRunners orders level clubs by the groups each had the
	// best runner in among them, then the second best, and so on.
	TieBreakBestRunners TieBreak = "best_runners"
)

// RulebookGroup is one group of the rulebook: a class name as the rules
// spell it, the points its scoring runners take in finishing order, and
// other names a competition uses for it.
type RulebookGroup struct {
	Name    string   `yaml:"name"`
	Scale   []int    `yaml:"scale"`
	Aliases []string `yaml:"aliases"`
}

// errRulebook marks every reason a rulebook file is refused.
var errRulebook = errors.New("divisionsmatch: invalid rulebook")

// Rulebooks is the versions embedded in the module, oldest first.
func Rulebooks() []string {
	entries, err := rulebookFiles.ReadDir("rulebooks")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".yml"))
	}
	slices.Sort(out)
	return out
}

// Embedded is the embedded rulebook of one version.
func Embedded(version string) (*Rulebook, error) {
	if !slices.Contains(Rulebooks(), version) {
		return nil, fmt.Errorf("%w: no embedded rulebook %q", errRulebook, version)
	}
	raw, err := rulebookFiles.ReadFile("rulebooks/" + version + ".yml")
	if err != nil {
		return nil, fmt.Errorf("divisionsmatch: read rulebook %q: %w", version, err)
	}
	return Parse(raw)
}

// Latest is the newest embedded rulebook. The embedded files are checked
// by the tests, so a broken one cannot ship, and Latest panics on one.
func Latest() *Rulebook {
	versions := Rulebooks()
	rb, err := Embedded(versions[len(versions)-1])
	if err != nil {
		panic(err)
	}
	return rb
}

// Load reads a rulebook file from disk: a new rulebook version can be
// tried before it is merged.
func Load(path string) (*Rulebook, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("divisionsmatch: read rulebook: %w", err)
	}
	return Parse(raw)
}

// Parse reads a rulebook from YAML and checks it. A field the schema does
// not know is an error, so a typo in a key cannot silently change the
// rules.
func Parse(raw []byte) (*Rulebook, error) {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	var rb Rulebook
	if err := dec.Decode(&rb); err != nil {
		return nil, fmt.Errorf("%w: %w", errRulebook, err)
	}
	if err := rb.validate(); err != nil {
		return nil, err
	}
	return &rb, nil
}

func (rb *Rulebook) validate() error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%w: "+format, append([]any{errRulebook}, args...)...))
	}
	if strings.TrimSpace(rb.Version) == "" {
		fail("version is empty")
	}
	mp := rb.MatchPoints
	if mp.Win <= mp.Loss || mp.Draw < mp.Loss || mp.Draw > mp.Win {
		fail("match_points must have win > loss and draw between them")
	}
	if rb.ClubLimit <= 0 || rb.ClubLimit > 1 {
		fail("club_limit must be more than 0 and at most 1")
	}
	if rb.Walkover.ScoreShare < 0 || rb.Walkover.ScoreShare > 1 {
		fail("walkover score_share must be from 0 to 1")
	}
	for i, tb := range rb.TieBreaks {
		if tb != TieBreakMutualScore && tb != TieBreakBestRunners {
			fail("tie_breaks: unknown tie-break %q", tb)
		}
		if slices.Contains(rb.TieBreaks[:i], tb) {
			fail("tie_breaks: %q is listed twice", tb)
		}
	}
	for _, s := range rb.ClassNames.SameGroupSuffixes {
		if groupKey(s) == "" {
			fail("same_group_suffixes: %q has no letters or digits", s)
		}
	}
	if len(rb.Groups) == 0 {
		fail("groups is empty")
	}
	keys := map[string]string{}
	for _, g := range rb.Groups {
		rb.validateGroup(g, keys, fail)
	}
	return errors.Join(errs...)
}

// validateGroup checks one group: a scale a club can score in, and names
// no other group answers to.
func (rb *Rulebook) validateGroup(g RulebookGroup, keys map[string]string, fail func(string, ...any)) {
	if strings.TrimSpace(g.Name) == "" {
		fail("a group has no name")
		return
	}
	if len(g.Scale) == 0 || slices.ContainsFunc(g.Scale, func(p int) bool { return p <= 0 }) {
		fail("group %s: scale must be points above 0", g.Name)
	} else if rb.perClub(g.Scale) < 1 {
		fail("group %s: club_limit gives no runner of the scale to a club", g.Name)
	}
	for _, name := range append([]string{g.Name}, g.Aliases...) {
		key := groupKey(name)
		if key == "" {
			fail("group %s: name %q has no letters or digits", g.Name, name)
			continue
		}
		if other, taken := keys[key]; taken {
			fail("group %s: name %q is also a name of group %s", g.Name, name, other)
			continue
		}
		keys[key] = g.Name
	}
}

// perClub is how many runners of one club score in a group with the
// scale given. The small margin keeps a share like 0.5 of 8 at 4 when
// the multiplication lands a hair under it.
func (rb *Rulebook) perClub(scale []int) int {
	return int(math.Floor(float64(len(scale))*rb.ClubLimit + scoreEpsilon))
}

// Group is a rulebook group by its name, false for a name the rulebook
// lacks. The scale is a copy.
func (rb *Rulebook) Group(name string) (RulebookGroup, bool) {
	i := slices.IndexFunc(rb.Groups, func(g RulebookGroup) bool { return g.Name == name })
	if i < 0 {
		return RulebookGroup{}, false
	}
	g := rb.Groups[i]
	return RulebookGroup{Name: g.Name, Scale: slices.Clone(g.Scale), Aliases: slices.Clone(g.Aliases)}, true
}

// GuessGroup pairs a competition's class with the rulebook group it is
// the same class as under another spelling: "H21-", "H-20", "D 50" and
// "H21-B" are how timing software and O-service write H21, H20, D50 and
// H21B. An alias of a group is the group, a class with one of the
// rulebook's same-group endings is the group without it ("H21A" is H21),
// and anything else is in no group: the organiser decides.
func (rb *Rulebook) GuessGroup(className string) string {
	key := groupKey(className)
	if key == "" {
		return ""
	}
	if name, ok := rb.groupByKey(key); ok {
		return name
	}
	for _, suffix := range rb.ClassNames.SameGroupSuffixes {
		if base, cut := strings.CutSuffix(key, groupKey(suffix)); cut {
			if name, ok := rb.groupByKey(base); ok {
				return name
			}
		}
	}
	return ""
}

func (rb *Rulebook) groupByKey(key string) (string, bool) {
	for _, g := range rb.Groups {
		if groupKey(g.Name) == key ||
			slices.ContainsFunc(g.Aliases, func(a string) bool { return groupKey(a) == key }) {
			return g.Name, true
		}
	}
	return "", false
}

// groupKey is a class name reduced to what identifies it: upper case,
// letters and digits only, so the dashes and spaces the spellings differ
// by fall away.
func groupKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalise puts a config in order: names trimmed, a division with no
// name and no clubs dropped, a club listed once, the absent clubs among
// the division's own, and a group the rulebook lacks read as no group. A
// config is usable after it whatever it held, and the one given is left
// as it was.
func (rb *Rulebook) normalise(c Config) Config {
	out := Config{Divisions: []Division{}, Groups: map[string]string{}}
	for _, d := range c.Divisions {
		d.Name = strings.TrimSpace(d.Name)
		d.Clubs = cleanNames(d.Clubs)
		if d.Name == "" && len(d.Clubs) == 0 {
			continue
		}
		absent := make([]string, 0, len(d.Absent))
		for _, club := range cleanNames(d.Absent) {
			if slices.Contains(d.Clubs, club) {
				absent = append(absent, club)
			}
		}
		d.Absent = absent
		out.Divisions = append(out.Divisions, d)
	}
	for class, group := range c.Groups {
		if _, ok := rb.Group(group); !ok {
			group = ""
		}
		out.Groups[class] = group
	}
	return out
}

// cleanNames is a list of names trimmed, without blanks, each once, in
// the order first given.
func cleanNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// compareScore compares two scores within the margin shares can leave.
func compareScore(a, b float64) int {
	if math.Abs(a-b) < scoreEpsilon {
		return 0
	}
	return cmp.Compare(a, b)
}
