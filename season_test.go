package divisionsmatch_test

import (
	"testing"

	dm "github.com/MetsaApp/divisionsmatch"
)

// A club that was in one round only has a zero line for the other, and
// its totals are those of the round it was in.
func TestSeasonClubInOneRound(t *testing.T) {
	t.Parallel()
	round := func(home, away string, hs, as float64, hp, ap int) dm.DivisionResult {
		return dm.DivisionResult{
			Standing: []dm.ClubStanding{
				{Club: home, MatchPoints: hp, ScoreFor: hs, ScoreAgainst: as},
				{Club: away, MatchPoints: ap, ScoreFor: as, ScoreAgainst: hs},
			},
			Matches: []dm.Match{{Home: home, Away: away, HomeScore: hs, AwayScore: as, HomePoints: hp, AwayPoints: ap}},
		}
	}
	s := dm.Latest().Season("1. division", []dm.DivisionResult{
		round("Alpha OK", "Beta OK", 120, 100, 2, 0),
		round("Alpha OK", "Gamma OK", 90, 110, 0, 2),
	})
	must(t, len(s.Standing) == 3, "season lines")
	var clubs []string
	for _, l := range s.Standing {
		clubs = append(clubs, l.Club)
	}
	// Alpha and Gamma have 2 match points each; Gamma won their match.
	equal(t, []string{"Gamma OK", "Alpha OK", "Beta OK"}, clubs, "season order")
	gamma := s.Standing[0]
	equal(t, dm.ClubStanding{Club: "Gamma OK"}, gamma.Rounds[0], "Gamma's round 1")
	totals := [3]float64{gamma.ScoreFor, gamma.ScoreAgainst, float64(gamma.MatchPoints)}
	equal(t, [3]float64{110, 90, 2}, totals, "Gamma's totals")
}
