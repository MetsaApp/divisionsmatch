package divisionsmatch

import (
	"cmp"
	"regexp"
	"slices"
)

// Result is a competition's divisionsmatch decided: every division of the
// config, computed from the runners as they stand.
type Result struct {
	Divisions []DivisionResult
}

// DivisionResult is one division decided: the standing in order, the
// matches as they are numbered, and the groups the points were taken in,
// in the rulebook's order, those with a runner from the division's clubs.
type DivisionResult struct {
	Name     string
	Standing []ClubStanding
	Matches  []Match
	Groups   []Group
}

// ClubStanding is one club's line of the standing: its match points, the
// score it took and the score taken against it over every match, and
// what separated it from a club level on match points, when something
// had to.
type ClubStanding struct {
	Club         string
	Absent       bool
	MatchPoints  int
	ScoreFor     float64
	ScoreAgainst float64
	// TiedScore is the club's score in its matches against the clubs level
	// with it, set with Tied when the mutual score tie-break was applied.
	// Placements are the counts of groups the club had the best, then the
	// second best, runner in among the clubs still level, set when the
	// best runners tie-break was applied.
	TiedScore  float64
	Tied       bool
	Placements []Placement
}

// Placement is a count of groups a club held one place in among the clubs
// it was level with: "3 groups where it had the best runner".
type Placement struct {
	Place int
	Count int
}

// Match is one pair of clubs: the score each took over every group and
// the match points that decided. An absent club scores nothing and takes
// the points for a loss; the club it failed to meet wins on a share of
// the score its runners would have earned.
type Match struct {
	Home, Away             string
	HomeAbsent, AwayAbsent bool
	HomeScore, AwayScore   float64
	HomePoints, AwayPoints int
}

// Group is one rulebook group of the division: its scale, its rows in
// finishing order with the points each took per match, and each match's
// score inside the group.
type Group struct {
	Name  string
	Scale []int
	Rows  []GroupRow
	// Scores is each match's score in this group, home then away, by
	// match index.
	Scores [][2]float64
}

// GroupRow is one runner of the division's clubs in the group, ranked at
// the finish among them, with the points they took per match. A runner
// who scored in no match has zeros; a runner running for a club of the
// division but outside the match (a running group's second runner, a
// fifth runner past the cap) has the same, and the row says why with
// Counted.
type GroupRow struct {
	Runner

	// Index is the runner's position in the slice given to Compute.
	Index int
	// Place is the runner's place among the division's runners in the
	// group, 0 for a runner with no place. Equal times share a place.
	Place int
	// Counted is whether the runner was in line for points: finished with
	// a place, from a club that turned up, and the first of their running
	// group to finish.
	Counted bool
	Points  []float64
}

// runningGroup is the rulebook's running group: runners entered together
// mark it in their names, "[g1]Anna Hansen", and only the fastest of
// them counts. The timing software has no field for it, so the name is
// where the mark is read from.
var runningGroup = regexp.MustCompile(`\[(.*?)\]`)

// scoreEpsilon is the margin two scores are equal within: shares of a
// scale are averages, which land a third away from a whole number.
const scoreEpsilon = 1e-6

// Compute decides every division of cfg from the competition's runners.
// A runner's class counts in the group cfg names for it, or in the group
// its name guesses when cfg names none; a class in no group is left out.
func (rb *Rulebook) Compute(cfg Config, runners []Runner) Result {
	cfg = rb.normalise(cfg)
	groups := rb.groupRunners(cfg, runners)
	var out Result
	for _, d := range cfg.Divisions {
		out.Divisions = append(out.Divisions, rb.newDivision(d, groups))
	}
	return out
}

// groupOf is the rulebook group a class counts in under cfg: the group
// cfg names for it, else the guess from its name.
func (rb *Rulebook) groupOf(cfg Config, class string) string {
	if g, saved := cfg.Groups[class]; saved {
		if _, ok := rb.Group(g); ok {
			return g
		}
		return ""
	}
	return rb.GuessGroup(class)
}

// groupRunners sorts the runners into their rulebook groups.
func (rb *Rulebook) groupRunners(cfg Config, runners []Runner) map[string][]indexed {
	groupOf := map[string]string{}
	out := map[string][]indexed{}
	for i, r := range runners {
		g, known := groupOf[r.Class]
		if !known {
			g = rb.groupOf(cfg, r.Class)
			groupOf[r.Class] = g
		}
		if g != "" {
			out[g] = append(out[g], indexed{Runner: r, index: i})
		}
	}
	return out
}

func (rb *Rulebook) newDivision(d Division, groups map[string][]indexed) DivisionResult {
	out := DivisionResult{Name: d.Name}
	absent := func(club string) bool { return slices.Contains(d.Absent, club) }
	for i, home := range d.Clubs {
		for _, away := range d.Clubs[i+1:] {
			out.Matches = append(out.Matches, Match{
				Home: home, Away: away, HomeAbsent: absent(home), AwayAbsent: absent(away),
			})
		}
	}
	for _, rg := range rb.Groups {
		rows := divisionRows(groups[rg.Name], d.Clubs)
		if len(rows) == 0 {
			continue
		}
		out.Groups = append(out.Groups, rb.newGroup(rg, rows, d.Absent, out.Matches))
	}
	for i := range out.Matches {
		m := &out.Matches[i]
		for _, g := range out.Groups {
			m.HomeScore += g.Scores[i][0]
			m.AwayScore += g.Scores[i][1]
		}
		m.HomePoints, m.AwayPoints = rb.matchPoints(*m)
	}
	out.Standing = rb.standing(d, out.Matches, out.Groups)
	return out
}

// divisionRows ranks the group's runners from the division's clubs at the
// finish among themselves.
func divisionRows(runners []indexed, clubs []string) []ranked {
	ours := make([]indexed, 0, len(runners))
	for _, r := range runners {
		if slices.Contains(clubs, r.Club) {
			ours = append(ours, r)
		}
	}
	if len(ours) == 0 {
		return nil
	}
	return rank(ours)
}

// newGroup gives every match its points in the group. The scoring runners
// are the placed rows of the clubs that turned up, the first of each
// running group only; a match reads them in finishing order and hands the
// scale out one point at a time to the runners whose club still has room.
// Runners on the same time who both took points share what they took
// between them.
func (rb *Rulebook) newGroup(rg RulebookGroup, rows []ranked, absent []string, matches []Match) Group {
	g := Group{Name: rg.Name, Scale: slices.Clone(rg.Scale), Scores: make([][2]float64, len(matches))}
	for _, r := range rows {
		g.Rows = append(g.Rows, GroupRow{
			Runner: r.Runner, Index: r.index, Place: r.place, Points: make([]float64, len(matches)),
		})
	}
	seen := map[string]bool{}
	for i := range g.Rows {
		row := &g.Rows[i]
		if row.Place == 0 || slices.Contains(absent, row.Club) {
			continue
		}
		if key := runningGroup.FindStringSubmatch(row.Name); key != nil {
			if seen[key[1]] {
				continue
			}
			seen[key[1]] = true
		}
		row.Counted = true
	}
	for mi, m := range matches {
		g.Scores[mi] = rb.scoreMatch(g.Rows, mi, m, rg.Scale)
	}
	return g
}

// scoreMatch hands the scale out for one match and sums each side.
func (rb *Rulebook) scoreMatch(rows []GroupRow, mi int, m Match, scale []int) [2]float64 {
	perClub := rb.perClub(scale)
	taken := map[string]int{}
	var scorers []*GroupRow
	given := 0
	for i := range rows {
		row := &rows[i]
		if !row.Counted {
			continue
		}
		in := (row.Club == m.Home && !m.HomeAbsent) || (row.Club == m.Away && !m.AwayAbsent)
		if !in || taken[row.Club] >= perClub {
			continue
		}
		if given < len(scale) {
			row.Points[mi] = float64(scale[given])
		}
		taken[row.Club]++
		given++
		scorers = append(scorers, row)
	}
	shareTies(scorers, mi)
	var score [2]float64
	for _, row := range scorers {
		if row.Club == m.Home {
			score[0] += row.Points[mi]
		} else {
			score[1] += row.Points[mi]
		}
	}
	if m.AwayAbsent {
		score[0] *= rb.Walkover.ScoreShare
	}
	if m.HomeAbsent {
		score[1] *= rb.Walkover.ScoreShare
	}
	return score
}

// shareTies averages the points of the scorers on one time: the scale
// gave them consecutive points in an order the clock did not decide.
func shareTies(scorers []*GroupRow, mi int) {
	for i := 0; i < len(scorers); {
		j := i + 1
		for j < len(scorers) && scorers[j].Time == scorers[i].Time {
			j++
		}
		if j-i > 1 {
			sum := 0.0
			for _, row := range scorers[i:j] {
				sum += row.Points[mi]
			}
			for _, row := range scorers[i:j] {
				row.Points[mi] = sum / float64(j-i)
			}
		}
		i = j
	}
}

// matchPoints decides a match: a win, a draw, or a walkover to the club
// that turned up.
func (rb *Rulebook) matchPoints(m Match) (int, int) {
	p := rb.MatchPoints
	switch {
	case m.HomeAbsent && m.AwayAbsent:
		return p.Loss, p.Loss
	case m.HomeAbsent:
		return p.Loss, p.Win
	case m.AwayAbsent:
		return p.Win, p.Loss
	case compareScore(m.HomeScore, m.AwayScore) > 0:
		return p.Win, p.Loss
	case compareScore(m.HomeScore, m.AwayScore) < 0:
		return p.Loss, p.Win
	default:
		return p.Draw, p.Draw
	}
}

// standing orders the clubs by match points, then separates the clubs
// level on them with the rulebook's tie-breaks in its order.
func (rb *Rulebook) standing(d Division, matches []Match, groups []Group) []ClubStanding {
	rows := make([]ClubStanding, 0, len(d.Clubs))
	for _, club := range d.Clubs {
		rows = append(rows, clubLine(club, slices.Contains(d.Absent, club), matches))
	}
	return rb.order(rows, matches, groups)
}

// order sorts club lines by match points, then separates the clubs level
// on them with the rulebook's tie-breaks in its order.
func (rb *Rulebook) order(rows []ClubStanding, matches []Match, groups []Group) []ClubStanding {
	slices.SortStableFunc(rows, func(a, b ClubStanding) int { return cmp.Compare(b.MatchPoints, a.MatchPoints) })
	out := make([]ClubStanding, 0, len(rows))
	for i := 0; i < len(rows); {
		j := i + 1
		for j < len(rows) && rows[j].MatchPoints == rows[i].MatchPoints {
			j++
		}
		out = append(out, breakTies(rows[i:j], rb.TieBreaks, matches, groups)...)
		i = j
	}
	return out
}

// breakTies orders clubs level so far with the first tie-break, and the
// clubs it leaves level with the ones after it. Clubs no tie-break
// separates keep the division's order.
func breakTies(level []ClubStanding, tieBreaks []TieBreak, matches []Match, groups []Group) []ClubStanding {
	if len(level) < 2 || len(tieBreaks) == 0 {
		return level
	}
	var parts [][]ClubStanding
	switch tieBreaks[0] {
	case TieBreakMutualScore:
		parts = byMutualScore(level, matches)
	case TieBreakBestRunners:
		parts = byPlacements(1, level, groups)
	default:
		parts = [][]ClubStanding{level}
	}
	out := make([]ClubStanding, 0, len(level))
	for _, p := range parts {
		out = append(out, breakTies(p, tieBreaks[1:], matches, groups)...)
	}
	return out
}

// clubLine sums one club's matches: its match points, and the score it
// took and had taken against it.
func clubLine(club string, absent bool, matches []Match) ClubStanding {
	s := ClubStanding{Club: club, Absent: absent}
	for _, m := range matches {
		switch club {
		case m.Home:
			s.MatchPoints += m.HomePoints
			s.ScoreFor += m.HomeScore
			s.ScoreAgainst += m.AwayScore
		case m.Away:
			s.MatchPoints += m.AwayPoints
			s.ScoreFor += m.AwayScore
			s.ScoreAgainst += m.HomeScore
		}
	}
	return s
}

// byMutualScore is the mutual score tie-break: each level club's score in
// its matches against the others of them, most first, the clubs on one
// score together.
func byMutualScore(level []ClubStanding, matches []Match) [][]ClubStanding {
	clubs := clubsOf(level)
	rows := slices.Clone(level)
	for i := range rows {
		s := &rows[i]
		s.Tied, s.TiedScore = true, 0
		for _, m := range matches {
			switch {
			case m.Home == s.Club && slices.Contains(clubs, m.Away):
				s.TiedScore += m.HomeScore
			case m.Away == s.Club && slices.Contains(clubs, m.Home):
				s.TiedScore += m.AwayScore
			}
		}
	}
	slices.SortStableFunc(rows, func(a, b ClubStanding) int { return compareScore(b.TiedScore, a.TiedScore) })
	var parts [][]ClubStanding
	for i := 0; i < len(rows); {
		j := i + 1
		for j < len(rows) && compareScore(rows[j].TiedScore, rows[i].TiedScore) == 0 {
			j++
		}
		parts = append(parts, rows[i:j])
		i = j
	}
	return parts
}

// byPlacements is the best runners tie-break: in every group, which of
// the level clubs the runner in the place named belongs to, among the
// placed runners of those clubs alone. The clubs with most such groups go
// first; clubs level on the count are separated by the next place down,
// and clubs with none of them come last, still level, in the order they
// were in.
func byPlacements(place int, level []ClubStanding, groups []Group) [][]ClubStanding {
	counts := placementCounts(place, level, groups)
	if len(counts) == 0 {
		return [][]ClubStanding{level}
	}
	var parts [][]ClubStanding
	for _, n := range countsDescending(counts) {
		var tied []ClubStanding
		for _, s := range level {
			if counts[s.Club] == n {
				s.Placements = append(slices.Clip(s.Placements), Placement{Place: place, Count: n})
				tied = append(tied, s)
			}
		}
		if len(tied) > 1 {
			parts = append(parts, byPlacements(place+1, tied, groups)...)
		} else {
			parts = append(parts, tied)
		}
	}
	var rest []ClubStanding
	for _, s := range level {
		if _, counted := counts[s.Club]; !counted {
			rest = append(rest, s)
		}
	}
	if len(rest) > 0 {
		parts = append(parts, rest)
	}
	return parts
}

// placementCounts counts, per club, the groups in which the club has the
// runner in the place named among the placed runners of the clubs given.
func placementCounts(place int, level []ClubStanding, groups []Group) map[string]int {
	clubs := clubsOf(level)
	counts := map[string]int{}
	for _, g := range groups {
		n := 0
		for _, row := range g.Rows {
			if row.Place == 0 || !slices.Contains(clubs, row.Club) {
				continue
			}
			n++
			if n == place {
				counts[row.Club]++
				break
			}
		}
	}
	return counts
}

func clubsOf(rows []ClubStanding) []string {
	out := make([]string, 0, len(rows))
	for _, s := range rows {
		out = append(out, s.Club)
	}
	return out
}

// countsDescending is the distinct counts, most first.
func countsDescending(counts map[string]int) []int {
	var out []int
	for _, n := range counts {
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	slices.SortFunc(out, func(a, b int) int { return cmp.Compare(b, a) })
	return out
}
