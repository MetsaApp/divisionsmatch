package divisionsmatch_test

import (
	"testing"

	dm "github.com/MetsaApp/divisionsmatch"
)

// The divisionsmatch is read through Compute, the one door: the config
// names the divisions and the groups, the rows are the competition's, and
// what comes out is every division's standing, its matches and its groups
// with the points each runner took in each match. The expected numbers
// are worked by hand from the rulebook, never from the code.

const (
	alpha = "Alpha OK"
	beta  = "Beta OK"
	gamma = "Gamma OK"
	delta = "Delta OK"

	classH21 = "H21-"
	classD21 = "D21-"
	classH50 = "H50-"
	classH16 = "H-16"
	classD50 = "D50-"
)

// ok is a finished runner with a time in whole seconds.
func ok(club, class, name string, seconds int) dm.Runner {
	return dm.Runner{Name: name, Club: club, Class: class, Status: dm.StatusOK, Time: seconds}
}

func twoClubs(name string) dm.Config {
	return dm.Config{Divisions: []dm.Division{{Name: name, Clubs: []string{alpha, beta}}}}
}

func rulebook(t *testing.T) *dm.Rulebook {
	t.Helper()
	rb, err := dm.Embedded("2025")
	if err != nil {
		t.Fatal(err)
	}
	return rb
}

// H21 scores eight runners, four per club at most, 8 down to 1 in the
// order the scoring runners finish. A fifth runner of a club takes nothing
// and holds no place on the scale, so the other club's next runner takes
// the point after the one last given. A runner without a result and a
// runner still out take nothing.
func TestPointsByFinishingOrder(t *testing.T) {
	runners := []dm.Runner{
		ok(alpha, classH21, "A1", 30*60),
		ok(beta, classH21, "B1", 31*60),
		ok(beta, classH21, "B2", 31*60+30),
		ok(alpha, classH21, "A2", 32*60),
		ok(alpha, classH21, "A3", 32*60+30),
		ok(alpha, classH21, "A4", 33*60),
		ok(alpha, classH21, "A5", 33*60+30),
		ok(beta, classH21, "B3", 34*60),
		{Name: "B4 MP", Club: beta, Class: classH21, Status: dm.StatusMP},
		{Name: "A6 out", Club: alpha, Class: classH21, Status: dm.StatusRunning},
	}
	res := rulebook(t).Compute(twoClubs("1. division"), runners)
	must(t, len(res.Divisions) == 1)
	d := res.Divisions[0]
	equal(t, "1. division", d.Name)

	must(t, len(d.Matches) == 1)
	m := d.Matches[0]
	equal(t, alpha, m.Home)
	equal(t, beta, m.Away)
	near(t, 8+5+4+3, m.HomeScore)
	near(t, 7+6+2, m.AwayScore)
	equal(t, 2, m.HomePoints)
	equal(t, 0, m.AwayPoints)

	must(t, len(d.Groups) == 1, "only the group with runners is listed")
	g := d.Groups[0]
	equal(t, "H21", g.Name)
	equal(t, []int{8, 7, 6, 5, 4, 3, 2, 1}, g.Scale)
	must(t, len(g.Rows) == 10)
	points := map[string]float64{}
	places := map[string]int{}
	for _, r := range g.Rows {
		points[r.Name] = r.Points[0]
		places[r.Name] = r.Place
	}
	equal(t, map[string]float64{
		"A1": 8, "B1": 7, "B2": 6, "A2": 5, "A3": 4, "A4": 3, "A5": 0, "B3": 2, "B4 MP": 0, "A6 out": 0,
	}, points)
	equal(t, 7, places["A5"], "a runner past the cap keeps their place in the group")
	equal(t, 0, places["B4 MP"])
	equal(t, [][2]float64{{20, 15}}, g.Scores)
	equal(t, 1, g.Rows[1].Index, "the row points back at the caller's slice")
	equal(t, "B4 MP", runners[g.Rows[9].Index].Name)

	must(t, len(d.Standing) == 2)
	equal(t, alpha, d.Standing[0].Club)
	equal(t, 2, d.Standing[0].MatchPoints)
	near(t, 20, d.Standing[0].ScoreFor)
	near(t, 15, d.Standing[0].ScoreAgainst)
	equal(t, beta, d.Standing[1].Club)
	equal(t, 0, d.Standing[1].MatchPoints)
}

// Runners on one time share the points the scale gave them, whichever club
// they run for: three level at the front of D21 take 8, 7 and 6 between
// them, 7 each. The rulebook reads times as a result list prints them, in
// whole seconds; a caller with tenths drops them before it calls.
func TestSharesPointsOnLevelTimes(t *testing.T) {
	runners := []dm.Runner{
		ok(alpha, classD21, "A1", 40*60),
		ok(beta, classD21, "B1", 40*60),
		ok(beta, classD21, "B2", 40*60),
		ok(alpha, classD21, "A2", 41*60),
		ok(alpha, classD21, "A3", 42*60),
		ok(beta, classD21, "B3", 42*60),
	}
	d := rulebook(t).Compute(twoClubs("1. division"), runners).Divisions[0]
	g := d.Groups[0]
	points := map[string]float64{}
	for _, r := range g.Rows {
		points[r.Name] = r.Points[0]
	}
	equal(t, map[string]float64{"A1": 7, "B1": 7, "B2": 7, "A2": 5, "A3": 3.5, "B3": 3.5}, points)
	near(t, 7+5+3.5, d.Matches[0].HomeScore)
	near(t, 7+7+3.5, d.Matches[0].AwayScore)
	equal(t, 0, d.Matches[0].HomePoints)
	equal(t, 2, d.Matches[0].AwayPoints)
	equal(t, 1, g.Rows[0].Place)
	equal(t, 1, g.Rows[2].Place, "three share first")
	equal(t, 4, g.Rows[3].Place)
	equal(t, 5, g.Rows[4].Place)
	equal(t, 5, g.Rows[5].Place, "one time is one place")
}

// A draw is a point each. With the scale {2, 2, 1, 1} of H14B, three
// level at the front share (2 + 2 + 1) / 3, which is no whole number, and
// the two scores are still read as equal.
func TestDrawOnThirds(t *testing.T) {
	const classH14B = "H14B"
	runners := []dm.Runner{
		ok(alpha, classH14B, "A1", 20*60),
		ok(beta, classH14B, "B1", 20*60),
		ok(beta, classH14B, "B2", 20*60),
		ok(alpha, classH14B, "A2", 21*60),
	}
	m := rulebook(t).Compute(twoClubs("3. division"), runners).Divisions[0].Matches[0]
	near(t, 5.0/3+1, m.HomeScore)
	near(t, 10.0/3, m.AwayScore)
	equal(t, 0, m.HomePoints, "2.67 against 3.33")
	equal(t, 2, m.AwayPoints)
}

// Runners entered as a running group carry its name in brackets, and only
// the fastest of them counts: the second takes nothing and holds no place
// on the scale, like a runner past the cap, and the row says so.
func TestRunningGroupCountsOnce(t *testing.T) {
	runners := []dm.Runner{
		ok(alpha, classH50, "Ann [g1]", 50*60),
		ok(alpha, classH50, "[g1] Bo", 51*60),
		ok(beta, classH50, "Cy", 52*60),
		ok(alpha, classH50, "Dan", 53*60),
	}
	g := rulebook(t).Compute(twoClubs("1. division"), runners).Divisions[0].Groups[0]
	equal(t, "H50", g.Name)
	equal(t, []float64{6}, g.Rows[0].Points)
	equal(t, true, g.Rows[0].Counted)
	equal(t, []float64{0}, g.Rows[1].Points)
	equal(t, false, g.Rows[1].Counted, "the second of the running group")
	equal(t, 2, g.Rows[1].Place, "but placed in the group all the same")
	equal(t, []float64{5}, g.Rows[2].Points)
	equal(t, []float64{4}, g.Rows[3].Points)
	equal(t, [][2]float64{{10, 5}}, g.Scores)
}

// A club that did not turn up loses every match on a walkover: its
// runners score nothing, the club it failed to meet takes two points on
// half the score its runners earn among themselves. Alpha alone in H21
// takes 8, 7, 6, 5 = 26, halved to 13.
func TestAbsentClub(t *testing.T) {
	cfg := dm.Config{Divisions: []dm.Division{
		{Name: "2. division", Clubs: []string{alpha, beta, gamma}, Absent: []string{gamma}},
	}}
	runners := []dm.Runner{
		ok(alpha, classH21, "A1", 30*60),
		ok(beta, classH21, "B1", 31*60),
		ok(beta, classH21, "B2", 31*60+30),
		ok(alpha, classH21, "A2", 32*60),
		ok(alpha, classH21, "A3", 32*60+30),
		ok(alpha, classH21, "A4", 33*60),
		ok(alpha, classH21, "A5", 33*60+30),
		ok(beta, classH21, "B3", 34*60),
		ok(gamma, classH21, "G1", 29*60),
	}
	d := rulebook(t).Compute(cfg, runners).Divisions[0]
	must(t, len(d.Matches) == 3)
	equal(t, [][2]float64{{20, 15}, {13, 0}, {10.5, 0}}, d.Groups[0].Scores)
	equal(t, []int{2, 2, 2}, []int{d.Matches[0].HomePoints, d.Matches[1].HomePoints, d.Matches[2].HomePoints})
	equal(t, []int{0, 0, 0}, []int{d.Matches[0].AwayPoints, d.Matches[1].AwayPoints, d.Matches[2].AwayPoints})
	equal(t, true, d.Matches[1].AwayAbsent)
	var g1 dm.GroupRow
	for _, r := range d.Groups[0].Rows {
		if r.Name == "G1" {
			g1 = r
		}
	}
	equal(t, 1, g1.Place, "the absent club's runner ran, and is placed")
	equal(t, false, g1.Counted)
	equal(t, []float64{0, 0, 0}, g1.Points)

	must(t, len(d.Standing) == 3)
	equal(t, []string{alpha, beta, gamma}, []string{d.Standing[0].Club, d.Standing[1].Club, d.Standing[2].Club})
	equal(t, []int{4, 2, 0}, []int{d.Standing[0].MatchPoints, d.Standing[1].MatchPoints, d.Standing[2].MatchPoints})
	near(t, 33, d.Standing[0].ScoreFor)
	near(t, 15, d.Standing[0].ScoreAgainst)
	near(t, 25.5, d.Standing[1].ScoreFor)
	equal(t, true, d.Standing[2].Absent)
}

// Clubs level on match points are separated by the score each took in the
// matches between them, not by their score over every match. Alpha beats
// all three; Beta, Gamma and Delta beat one another round in a circle on
// three groups, each the home ground of one pair:
//
//	H16 (4 3 2 1): A1 B1 B2 A2 C1 C2            Beta beats Gamma 7-3
//	D50 (6..1):    A A A C C C D D D            Gamma beats Delta 15-6
//	H50 (6..1):    A D D D B B B A A            Delta beats Beta 15-6
//
// Over the three groups Beta beats Gamma 22-18, Gamma beats Delta 22-21
// and Delta beats Beta 30-13. Among the three, Delta took 51, Gamma 40 and
// Beta 35, so that is the order, although Beta's score over every match
// (52, with 17 against Alpha) beats Gamma's (49).
func TestStandingTiedOnMutualScore(t *testing.T) {
	cfg := dm.Config{Divisions: []dm.Division{{Name: "1. division", Clubs: []string{alpha, beta, gamma, delta}}}}
	var runners []dm.Runner
	add := func(class string, base int, order ...string) {
		for i, club := range order {
			runners = append(runners, ok(club, class, "R", (base*10+i)*60))
		}
	}
	add(classH16, 13, alpha, beta, beta, alpha, gamma, gamma)
	add(classD50, 14, alpha, alpha, alpha, gamma, gamma, gamma, delta, delta, delta)
	add(classH50, 12, alpha, delta, delta, delta, beta, beta, beta, alpha, alpha)
	d := rulebook(t).Compute(cfg, runners).Divisions[0]

	scores := map[[2]string][2]float64{}
	for _, m := range d.Matches {
		scores[[2]string{m.Home, m.Away}] = [2]float64{m.HomeScore, m.AwayScore}
	}
	equal(t, map[[2]string][2]float64{
		{alpha, beta}: {29, 17}, {alpha, gamma}: {37, 9}, {alpha, delta}: {31, 18},
		{beta, gamma}: {22, 18}, {beta, delta}: {13, 30}, {gamma, delta}: {22, 21},
	}, scores)

	order := make([]string, 0, 4)
	for _, s := range d.Standing {
		order = append(order, s.Club)
	}
	equal(t, []string{alpha, delta, gamma, beta}, order)
	equal(t, 6, d.Standing[0].MatchPoints)
	equal(t, false, d.Standing[0].Tied)
	near(t, 97, d.Standing[0].ScoreFor)
	for _, s := range d.Standing[1:] {
		equal(t, 2, s.MatchPoints)
		equal(t, true, s.Tied)
		equal(t, 0, len(s.Placements), "the mutual score separated them")
	}
	near(t, 51, d.Standing[1].TiedScore)
	near(t, 40, d.Standing[2].TiedScore)
	near(t, 35, d.Standing[3].TiedScore)
	near(t, 52, d.Standing[3].ScoreFor)
}

// Two clubs that draw their one match are level on points and on mutual
// score by definition, so the groups decide: Alpha has the best runner in
// two of the three groups, Beta in one.
func TestStandingTiedOnBestRunners(t *testing.T) {
	runners := []dm.Runner{
		ok(alpha, classH16, "A", 10*60), ok(beta, classH16, "B", 11*60),
		ok(beta, classD50, "B", 20*60), ok(alpha, classD50, "A", 21*60),
		ok(alpha, classH21, "A", 30*60), ok(beta, classH21, "B", 31*60),
		ok(beta, classH21, "B", 32*60), ok(alpha, classH21, "A", 33*60),
	}
	d := rulebook(t).Compute(twoClubs("1. division"), runners).Divisions[0]
	m := d.Matches[0]
	near(t, 4+5+13, m.HomeScore)
	near(t, 3+6+13, m.AwayScore)
	equal(t, 1, m.HomePoints)
	equal(t, 1, m.AwayPoints)
	must(t, len(d.Standing) == 2)
	equal(t, alpha, d.Standing[0].Club)
	equal(t, []dm.Placement{{Place: 1, Count: 2}}, d.Standing[0].Placements)
	equal(t, beta, d.Standing[1].Club)
	equal(t, []dm.Placement{{Place: 1, Count: 1}}, d.Standing[1].Placements)
	equal(t, true, d.Standing[0].Tied)
	near(t, 22, d.Standing[0].TiedScore)
}

// The tie-breaks are the rulebook's, in its order. A rulebook that lists
// only the best runners decides the circle of the mutual score test on
// best runners alone: the order changes, and no club is marked tied on
// score.
func TestTieBreakOrderIsData(t *testing.T) {
	rb := rulebook(t)
	rb.TieBreaks = []dm.TieBreak{dm.TieBreakBestRunners}
	runners := []dm.Runner{
		ok(alpha, classH16, "A", 10*60), ok(beta, classH16, "B", 11*60),
		ok(beta, classD50, "B", 20*60), ok(alpha, classD50, "A", 21*60),
		ok(alpha, classH21, "A", 30*60), ok(beta, classH21, "B", 31*60),
		ok(beta, classH21, "B", 32*60), ok(alpha, classH21, "A", 33*60),
	}
	d := rb.Compute(twoClubs("1. division"), runners).Divisions[0]
	equal(t, alpha, d.Standing[0].Club)
	equal(t, false, d.Standing[0].Tied)
	equal(t, []dm.Placement{{Place: 1, Count: 2}}, d.Standing[0].Placements)

	rb.TieBreaks = nil
	d = rb.Compute(twoClubs("1. division"), runners).Divisions[0]
	equal(t, []string{alpha, beta}, []string{d.Standing[0].Club, d.Standing[1].Club}, "no tie-break keeps the order")
	equal(t, 0, len(d.Standing[0].Placements))
}

// Two divisions in one competition are two matches on the same courses:
// each reads the runners of its own clubs and nobody else's. A class the
// guess leaves out counts once the organiser names a group for it, and a
// runner with no final result yet is not placed.
func TestTwoDivisionsAndSavedGroups(t *testing.T) {
	const classOpen = "Åben 1"
	cfg := dm.Config{Divisions: []dm.Division{
		{Name: "1. division", Clubs: []string{alpha, beta}},
		{Name: "2. division", Clubs: []string{gamma, delta}},
	}}
	prel := ok(beta, classH21, "B prel", 28*60)
	prel.Status = dm.StatusRunning
	runners := []dm.Runner{
		ok(alpha, classH21, "A1", 30*60),
		ok(gamma, classH21, "G1", 30*60+30),
		ok(beta, classH21, "B1", 31*60),
		ok(delta, classH21, "D1", 31*60+30),
		prel,
		ok(alpha, classOpen, "E1", 29*60),
	}
	rb := rulebook(t)
	res := rb.Compute(cfg, runners)
	must(t, len(res.Divisions) == 2)
	first, second := res.Divisions[0], res.Divisions[1]
	names := func(g dm.Group) []string {
		var out []string
		for _, r := range g.Rows {
			out = append(out, r.Name)
		}
		return out
	}
	equal(t, []string{"A1", "B1", "B prel"}, names(first.Groups[0]), "a provisional row is still running")
	equal(t, [][2]float64{{8, 7}}, first.Groups[0].Scores)
	equal(t, []string{"G1", "D1"}, names(second.Groups[0]))
	equal(t, [][2]float64{{8, 7}}, second.Groups[0].Scores)
	equal(t, "2. division", second.Name)

	cfg.Groups = map[string]string{classOpen: "H21"}
	first = rb.Compute(cfg, runners).Divisions[0]
	equal(t, []string{"E1", "A1", "B1", "B prel"}, names(first.Groups[0]))
	equal(t, [][2]float64{{15, 6}}, first.Groups[0].Scores)
	equal(t, false, first.Groups[0].Rows[3].Counted)

	cfg.Groups = map[string]string{classH21: ""}
	first = rb.Compute(cfg, runners).Divisions[0]
	equal(t, 0, len(first.Groups), "a class named into no group is out")
}
