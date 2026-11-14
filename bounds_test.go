package divisionsmatch_test

import (
	"math/rand/v2"
	"path/filepath"
	"testing"

	dm "github.com/MetsaApp/divisionsmatch"
)

// While runners are out, every match has a best and a worst score left for
// each side: the best when the side's runners still out all finish ahead
// of everyone and the other side's all fail, the worst the other way
// round. A match is decided when neither extreme changes who wins.

// running is an H21 runner with no result yet.
func running(club, name string) dm.Runner {
	return dm.Runner{Name: name, Club: club, Class: classH21, Status: dm.StatusRunning}
}

// Alpha leads 15-6 in H21 with one Beta runner out. If that runner wins,
// the scale gives Beta 8+5 and Alpha 7+6: 13-13, so a draw is still on.
func TestBoundsDrawStillPossible(t *testing.T) {
	runners := []dm.Runner{
		ok(alpha, classH21, "A1", 30*60),
		ok(alpha, classH21, "A2", 31*60),
		ok(beta, classH21, "B1", 32*60),
		running(beta, "B2"),
	}
	d := rulebook(t).Compute(twoClubs("1. division"), runners).Divisions[0]
	m := d.Matches[0]
	near(t, 15, m.HomeScore, "home now")
	near(t, 6, m.AwayScore, "away now")
	near(t, 15, m.HomeBest, "home best: the Beta runner fails")
	near(t, 13, m.HomeWorst, "home worst: the Beta runner wins")
	near(t, 13, m.AwayBest, "away best")
	near(t, 6, m.AwayWorst, "away worst")
	equal(t, false, m.Decided, "a draw is still possible")
	equal(t, 0, d.Standing[0].Running, "alpha has nobody out")
	equal(t, 1, clubLine(d, beta).Running, "beta has one out")
}

// With a third Alpha runner ahead of Beta's first, the best Beta can do
// is 8+4 against 7+6+5: Alpha has won whatever happens.
func TestBoundsDecided(t *testing.T) {
	runners := []dm.Runner{
		ok(alpha, classH21, "A1", 30*60),
		ok(alpha, classH21, "A2", 31*60),
		ok(alpha, classH21, "A3", 31*60+30),
		ok(beta, classH21, "B1", 32*60),
		running(beta, "B2"),
	}
	m := rulebook(t).Compute(twoClubs("1. division"), runners).Divisions[0].Matches[0]
	near(t, 18, m.HomeWorst, "home worst")
	near(t, 12, m.AwayBest, "away best")
	equal(t, true, m.Decided)
}

// Nobody out: the bounds are the score and every match is decided, a draw
// included.
func TestBoundsAllFinished(t *testing.T) {
	runners := []dm.Runner{
		ok(alpha, classH21, "A1", 30*60),
		ok(beta, classH21, "B1", 30*60),
	}
	m := rulebook(t).Compute(twoClubs("1. division"), runners).Divisions[0].Matches[0]
	near(t, m.HomeScore, m.HomeBest, "home best")
	near(t, m.HomeScore, m.HomeWorst, "home worst")
	near(t, m.AwayScore, m.AwayBest, "away best")
	near(t, m.AwayScore, m.AwayWorst, "away worst")
	equal(t, 1, m.HomePoints, "a draw")
	equal(t, true, m.Decided)
}

// A walkover is decided before anyone runs: the club that turned up wins,
// and an absent club's runners are not out for it.
func TestBoundsWalkover(t *testing.T) {
	cfg := dm.Config{Divisions: []dm.Division{
		{Name: "2. division", Clubs: []string{alpha, beta}, Absent: []string{beta}},
	}}
	runners := []dm.Runner{running(alpha, "A1"), running(beta, "B1")}
	d := rulebook(t).Compute(cfg, runners).Divisions[0]
	equal(t, true, d.Matches[0].Decided)
	equal(t, 1, clubLine(d, alpha).Running)
	equal(t, 0, clubLine(d, beta).Running, "an absent club has nobody out")
}

// The golden races, with a random share of the results not in yet: the
// final score lies inside every bound, and a match decided early ends the
// way it stood.
func TestBoundsHoldOnGoldenRaces(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*", "race.yml"))
	if err != nil {
		t.Fatal(err)
	}
	must(t, len(files) > 0, "no golden race")
	rb := rulebook(t)
	for _, file := range files {
		t.Run(filepath.Base(filepath.Dir(file)), func(t *testing.T) {
			dir := filepath.Dir(file)
			race := readRace(t, file)
			cfg := raceConfig(race)
			all := raceRunners(t, dir, race)
			final := rb.Compute(cfg, all)
			rng := rand.New(rand.NewPCG(1, uint64(len(all))))
			for _, share := range []float64{0.1, 0.3, 0.5, 0.8, 0.95} {
				for range 4 {
					partial := make([]dm.Runner, len(all))
					copy(partial, all)
					for i := range partial {
						if rng.Float64() < share {
							partial[i].Status, partial[i].Time = dm.StatusRunning, 0
						}
					}
					checkBounds(t, share, final, rb.Compute(cfg, partial))
				}
			}
		})
	}
}

func checkBounds(t *testing.T, share float64, final, now dm.Result) {
	t.Helper()
	const eps = 1e-6
	for di, d := range now.Divisions {
		for mi, m := range d.Matches {
			f := final.Divisions[di].Matches[mi]
			if f.HomeScore < m.HomeWorst-eps || f.HomeScore > m.HomeBest+eps ||
				f.AwayScore < m.AwayWorst-eps || f.AwayScore > m.AwayBest+eps {
				t.Errorf("%.0f%% out, %s %s-%s: final %v-%v outside [%v,%v]-[%v,%v]",
					share*100, d.Name, m.Home, m.Away, f.HomeScore, f.AwayScore,
					m.HomeWorst, m.HomeBest, m.AwayWorst, m.AwayBest)
			}
			if m.Decided && (f.HomePoints != m.HomePoints || f.AwayPoints != m.AwayPoints) {
				t.Errorf("%.0f%% out, %s %s-%s: decided at %d-%d, ended %d-%d",
					share*100, d.Name, m.Home, m.Away, m.HomePoints, m.AwayPoints, f.HomePoints, f.AwayPoints)
			}
		}
	}
}

func clubLine(d dm.DivisionResult, club string) dm.ClubStanding {
	for _, s := range d.Standing {
		if s.Club == club {
			return s
		}
	}
	return dm.ClubStanding{}
}
