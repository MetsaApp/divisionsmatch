package divisionsmatch_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/MetsaApp/iofxml"
	"gopkg.in/yaml.v3"

	dm "github.com/MetsaApp/divisionsmatch"
)

// A golden race is a real competition with its official result: a folder
// in testdata with the result list as IOF XML, the text of the official
// printouts, and a race.yml that names the
// divisions and holds the standing and match scores copied by hand from
// the printouts. Every folder with a race.yml is a test, so a new race
// needs no Go.
//
// The test never reads a runner's name: rows are matched by club, class
// and time, so the names can be anything.

// goldenRace is a race.yml.
type goldenRace struct {
	Results   string           `yaml:"results"`
	Divisions []goldenDivision `yaml:"divisions"`
}

type goldenDivision struct {
	Name     string `yaml:"name"`
	Official string `yaml:"official"`
	// Clubs are in match order.
	Clubs []goldenClub `yaml:"clubs"`
	// Standing and Matches are copied by hand from the printout.
	Standing []goldenStanding `yaml:"standing"`
	Matches  [][2]float64     `yaml:"matches"`
}

// goldenClub is a club as the result list names it and as the official
// printout names it: the printout uses the federation's club names, which
// are not always the timing software's.
type goldenClub struct {
	Name    string `yaml:"name"`
	Printed string `yaml:"printed"`
}

type goldenStanding struct {
	Club         string  `yaml:"club"`
	ScoreFor     float64 `yaml:"score"`
	ScoreAgainst float64 `yaml:"against"`
	MatchPoints  int     `yaml:"match_points"`
}

// TestGoldenRaces decides every golden race from its result list and
// compares the result with the official printouts: every standing line,
// every match score, every cell of the group-by-match grid, and every
// runner row's place and points.
func TestGoldenRaces(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*", "race.yml"))
	if err != nil {
		t.Fatal(err)
	}
	must(t, len(files) > 0, "no golden race")
	for _, file := range files {
		t.Run(filepath.Base(filepath.Dir(file)), func(t *testing.T) { testGoldenRace(t, file) })
	}
}

func testGoldenRace(t *testing.T, file string) {
	t.Helper()
	dir := filepath.Dir(file)
	race := readRace(t, file)
	xmlFile, err := os.Open(filepath.Join(dir, race.Results))
	if err != nil {
		t.Fatal(err)
	}
	defer xmlFile.Close()
	doc, err := iofxml.Read(xmlFile)
	if err != nil {
		t.Fatal(err)
	}
	must(t, doc.ResultList != nil, "results is no ResultList: ", doc.Kind)
	runners := runnersOf(doc.ResultList)
	cfg := dm.Config{}
	for _, g := range race.Divisions {
		d := dm.Division{Name: g.Name}
		for _, c := range g.Clubs {
			d.Clubs = append(d.Clubs, c.Name)
		}
		cfg.Divisions = append(cfg.Divisions, d)
	}
	rb := rulebook(t)
	res := rb.Compute(cfg, runners)
	must(t, len(res.Divisions) == len(race.Divisions))
	for i, g := range race.Divisions {
		t.Run(filepath.Base(g.Official), func(t *testing.T) {
			raw, readErr := os.ReadFile(filepath.Join(dir, g.Official))
			if readErr != nil {
				t.Fatal(readErr)
			}
			official := parsePrintout(t, string(raw), g, rb)
			checkTranscription(t, g, official)
			n := checkDivision(t, g, official, res.Divisions[i], rb)
			t.Logf("%d standing lines, %d matches, %d grid cells, %d runner rows agree",
				len(g.Standing), len(g.Matches), n.cells, n.rows)
		})
	}
}

// readRace reads a race.yml, with a club's printed name defaulting to its
// name.
func readRace(t *testing.T, file string) goldenRace {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	var race goldenRace
	if err = dec.Decode(&race); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	for i := range race.Divisions {
		for j := range race.Divisions[i].Clubs {
			c := &race.Divisions[i].Clubs[j]
			if c.Printed == "" {
				c.Printed = c.Name
			}
		}
	}
	return race
}

// printout is what the test reads from an official printout, with the
// clubs under the result list's names.
type printout struct {
	standing []goldenStanding
	matches  []printedMatch
	// grid is each group's score per match, Ialt the totals.
	grid map[string][][2]float64
	ialt [][2]float64
	rows []printedRow
}

type printedMatch struct {
	home, away string
	score      [2]float64
}

type printedRow struct {
	group, club, class string
	// place and time are 0 for a runner without a place; status is then
	// the printout's word for why.
	place, time int
	status      string
	points      []float64
}

var (
	reStandingHeader = regexp.MustCompile(`^\s+score\s+point\s*$`)
	reStanding       = regexp.MustCompile(`^(\S.*?)\s+(\d+(?:\.\d)?)\s+-\s+(\d+(?:\.\d)?)\s+(\d+)\s*$`)
	reMatch          = regexp.MustCompile(`^(\d+) : (.+?)\s+:\s+(\d+(?:\.\d)?)\s+-\s+(\d+(?:\.\d)?)\s*$`)
	reGrid           = regexp.MustCompile(`^(\S+)\s+((?:\d+(?:\.\d)?-\d+(?:\.\d)?\s*)+)$`)
	reCell           = regexp.MustCompile(`(\d+(?:\.\d)?)-(\d+(?:\.\d)?)`)
	reGroupHeader    = regexp.MustCompile(`^(\S+) - \[[\d, ]+\]$`)
	rePlaced         = regexp.MustCompile(`^(\d+)\s+(.*?)\s(\S+)\s+(\d\d):(\d\d):(\d\d)((?:\s+\d+(?:\.\d)?)*)\s*$`)
	reUnplaced       = regexp.MustCompile(`^\s+(.*?)\s+(Fejlst|Ej start|Ej slut|Disk|Maxtid)\s*$`)
)

func parsePrintout(t *testing.T, text string, g goldenDivision, rb *dm.Rulebook) printout {
	t.Helper()
	p := printout{grid: map[string][][2]float64{}}
	inStanding, group := false, ""
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimLeft(line, "\f")
		switch {
		case reStandingHeader.MatchString(line):
			inStanding = true
		case inStanding && strings.TrimSpace(line) == "":
			inStanding = false
		case inStanding:
			m := reStanding.FindStringSubmatch(line)
			must(t, m != nil, "standing line: ", line)
			p.standing = append(p.standing, goldenStanding{
				Club: clubByPrinted(t, g, m[1]), ScoreFor: num(t, m[2]), ScoreAgainst: num(t, m[3]),
				MatchPoints: int(num(t, m[4])),
			})
		case reMatch.MatchString(line):
			m := reMatch.FindStringSubmatch(line)
			home, away := matchClubs(t, g, m[2])
			p.matches = append(p.matches, printedMatch{home, away, [2]float64{num(t, m[3]), num(t, m[4])}})
		case reGroupHeader.MatchString(line):
			group = reGroupHeader.FindStringSubmatch(line)[1]
		case reGrid.MatchString(line):
			m := reGrid.FindStringSubmatch(line)
			var cells [][2]float64
			for _, c := range reCell.FindAllStringSubmatch(m[2], -1) {
				cells = append(cells, [2]float64{num(t, c[1]), num(t, c[2])})
			}
			if m[1] == "Ialt:" {
				p.ialt = cells
			} else if _, ok := rb.Group(m[1]); ok {
				p.grid[m[1]] = cells
			}
		case group != "" && rePlaced.MatchString(line):
			m := rePlaced.FindStringSubmatch(line)
			row := printedRow{
				group: group, club: clubInRow(t, g, m[2]), class: m[3], place: int(num(t, m[1])),
				time: int(num(t, m[4])*3600 + num(t, m[5])*60 + num(t, m[6])),
			}
			for tok := range strings.FieldsSeq(m[7]) {
				row.points = append(row.points, num(t, tok))
			}
			p.rows = append(p.rows, row)
		case group != "" && reUnplaced.MatchString(line):
			m := reUnplaced.FindStringSubmatch(line)
			p.rows = append(p.rows, printedRow{group: group, club: clubInRow(t, g, m[1]), status: m[2]})
		}
	}
	return p
}

// checkTranscription holds the hand-transcribed numbers against the
// printout they were transcribed from.
func checkTranscription(t *testing.T, g goldenDivision, p printout) {
	t.Helper()
	equal(t, g.Standing, p.standing, "the transcribed standing is the printout's")
	must(t, len(p.matches) == len(g.Matches), "matches in the printout")
	for i, m := range p.matches {
		equal(t, g.Matches[i], m.score, "transcribed match ", i+1)
	}
}

type agreed struct{ cells, rows int }

func checkDivision(t *testing.T, g goldenDivision, p printout, d dm.DivisionResult, rb *dm.Rulebook) agreed {
	t.Helper()
	var n agreed
	must(t, len(d.Standing) == len(g.Standing), "standing lines")
	for i, want := range g.Standing {
		got := d.Standing[i]
		equal(t, want.Club, got.Club, "standing ", i+1)
		equal(t, want.MatchPoints, got.MatchPoints, "match points of ", want.Club)
		near(t, want.ScoreFor, got.ScoreFor, "score of ", want.Club)
		near(t, want.ScoreAgainst, got.ScoreAgainst, "score against ", want.Club)
	}
	must(t, len(d.Matches) == len(p.matches), "matches")
	for i, m := range d.Matches {
		equal(t, p.matches[i].home+" - "+p.matches[i].away, m.Home+" - "+m.Away, "match order ", i+1)
		equal(t, g.Matches[i], [2]float64{m.HomeScore, m.AwayScore}, "match ", i+1)
	}
	// The grid has every rulebook group; a group the division has no
	// runner in is 0-0 in every match.
	must(t, len(p.grid) == len(rb.Groups), "the grid lists every rulebook group")
	scores := map[string][][2]float64{}
	for _, gr := range d.Groups {
		scores[gr.Name] = gr.Scores
	}
	for _, rg := range rb.Groups {
		want := p.grid[rg.Name]
		must(t, len(want) == len(d.Matches), "grid row ", rg.Name)
		for mi := range d.Matches {
			got := [2]float64{}
			if s, ok := scores[rg.Name]; ok {
				got = s[mi]
			}
			equal(t, want[mi], got, rg.Name, " m", mi+1)
			n.cells++
		}
	}
	for mi, m := range d.Matches {
		equal(t, p.ialt[mi], [2]float64{m.HomeScore, m.AwayScore}, "Ialt m", mi+1)
		n.cells++
	}
	n.rows = checkRows(t, p, d)
	return n
}

// checkRows matches every printed runner row to one row of the result
// by group, club, class and time, and compares place and points. A
// runner's points are printed in the matches of their club only, so the
// printed numbers are read against those matches in order.
func checkRows(t *testing.T, p printout, d dm.DivisionResult) int {
	t.Helper()
	type key struct{ group, club, class, status string }
	var unmatched []string
	pending := map[key][]dm.GroupRow{}
	groupNames := []string{}
	for _, gr := range d.Groups {
		groupNames = append(groupNames, gr.Name)
		for _, r := range gr.Rows {
			k := key{gr.Name, r.Club, r.Class, printedStatus(r)}
			if k.status != "" {
				k.class = ""
			}
			pending[k] = append(pending[k], r)
		}
	}
	printedGroups := []string{}
	for _, row := range p.rows {
		if !slices.Contains(printedGroups, row.group) {
			printedGroups = append(printedGroups, row.group)
		}
		k := key{row.group, row.club, row.class, row.status}
		if row.status != "" {
			k.class = ""
		}
		i := slices.IndexFunc(pending[k], func(r dm.GroupRow) bool {
			return row.status != "" || (r.Place == row.place && r.Time == row.time)
		})
		if i < 0 {
			unmatched = append(unmatched, fmt.Sprintf("%+v", row))
			continue
		}
		got := pending[k][i]
		pending[k] = slices.Delete(pending[k], i, i+1)
		checkPoints(t, row, got, d.Matches)
	}
	for k, rest := range pending {
		if len(rest) > 0 {
			unmatched = append(unmatched, fmt.Sprintf("result rows not in the printout: %+v x%d", k, len(rest)))
		}
	}
	for _, u := range unmatched {
		t.Error("unmatched: ", u)
	}
	equal(t, groupNames, printedGroups, "the groups with rows")
	return len(p.rows)
}

// checkPoints compares a row's points in its club's matches with the
// printed numbers, and checks it took none in the other matches.
func checkPoints(t *testing.T, row printedRow, got dm.GroupRow, matches []dm.Match) {
	t.Helper()
	var printed []float64
	for mi, m := range matches {
		if m.Home == row.club || m.Away == row.club {
			printed = append(printed, got.Points[mi])
		} else if got.Points[mi] != 0 {
			t.Errorf("%s %s: points in m%d, a match of other clubs", row.group, row.club, mi+1)
		}
	}
	want := row.points
	if len(want) == 0 {
		want = make([]float64, len(printed))
	}
	equal(t, want, printed, "points of ", fmt.Sprintf("%+v", row))
}

// printedStatus is the printout's word for a row without a place, and
// empty for a placed row. The class is left out of the key for those
// rows, so it is blanked here too.
func printedStatus(r dm.GroupRow) string {
	if r.Place > 0 {
		return ""
	}
	switch r.Status { //nolint:exhaustive // Only the statuses the printout has a word for.
	case dm.StatusMP:
		return "Fejlst"
	case dm.StatusDNS:
		return "Ej start"
	case dm.StatusDNF:
		return "Ej slut"
	case dm.StatusDSQ:
		return "Disk"
	case dm.StatusOT:
		return "Maxtid"
	default:
		return r.Status.String()
	}
}

func clubByPrinted(t *testing.T, g goldenDivision, printed string) string {
	t.Helper()
	for _, c := range g.Clubs {
		if c.Printed == printed {
			return c.Name
		}
	}
	t.Fatalf("no club printed as %q", printed)
	return ""
}

// matchClubs reads "home - away" with the printed club names.
func matchClubs(t *testing.T, g goldenDivision, pair string) (string, string) {
	t.Helper()
	for _, h := range g.Clubs {
		for _, a := range g.Clubs {
			if pair == h.Printed+" - "+a.Printed {
				return h.Name, a.Name
			}
		}
	}
	t.Fatalf("no clubs in match %q", pair)
	return "", ""
}

// clubInRow finds the club in a runner row, where the printout cuts the
// club's name to 15 characters.
func clubInRow(t *testing.T, g goldenDivision, text string) string {
	t.Helper()
	found := ""
	for _, c := range g.Clubs {
		cut := []rune(c.Printed)
		if len(cut) > 15 {
			cut = cut[:15]
		}
		if strings.Contains(text, string(cut)) {
			must(t, found == "", "two clubs in row ", text)
			found = c.Name
		}
	}
	must(t, found != "", "no club in row ", text)
	return found
}

func num(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
