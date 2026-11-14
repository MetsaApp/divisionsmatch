package divisionsmatch

import "slices"

// The bounds of a match while runners are out. A club does best in every
// one of its matches at once when its runners still out all finish ahead
// of everyone and every other club's runners still out fail: a runner who
// finishes can only push the other club's runners down the scale, and can
// never take points from their own club, because what they take was
// handed to a slower runner before. So one run of the division per club,
// in that club's best case, gives the club's best score and its
// opponents' worst in every match, with the rules themselves doing the
// counting. Ties between finishers only share points out, so they move no
// bound.

// withBounds sets the bounds, Decided and the clubs' Running on a
// division decided as the runners stand.
func (rb *Rulebook) withBounds(res DivisionResult, d Division, groups map[string][]indexed) DivisionResult {
	running := rb.runningOf(d, groups)
	for i := range res.Matches {
		m := &res.Matches[i]
		m.HomeBest, m.HomeWorst = m.HomeScore, m.HomeScore
		m.AwayBest, m.AwayWorst = m.AwayScore, m.AwayScore
	}
	for i := range res.Standing {
		res.Standing[i].Running = running[res.Standing[i].Club]
	}
	if len(running) > 0 {
		for _, club := range d.Clubs {
			best := rb.newDivision(d, bestCase(groups, d, club))
			for i := range res.Matches {
				m, b := &res.Matches[i], best.Matches[i]
				switch club {
				case m.Home:
					m.HomeBest, m.AwayWorst = b.HomeScore, b.AwayScore
				case m.Away:
					m.AwayBest, m.HomeWorst = b.AwayScore, b.HomeScore
				}
			}
		}
	}
	for i := range res.Matches {
		m := &res.Matches[i]
		hb, ab := rb.matchPoints(Match{
			HomeAbsent: m.HomeAbsent, AwayAbsent: m.AwayAbsent, HomeScore: m.HomeBest, AwayScore: m.AwayWorst,
		})
		hw, aw := rb.matchPoints(Match{
			HomeAbsent: m.HomeAbsent, AwayAbsent: m.AwayAbsent, HomeScore: m.HomeWorst, AwayScore: m.AwayBest,
		})
		m.Decided = hb == hw && ab == aw
	}
	return res
}

// runningOf counts, per club of the division that turned up, its runners
// in the groups with no final result yet. Clubs with none are left out.
func (rb *Rulebook) runningOf(d Division, groups map[string][]indexed) map[string]int {
	out := map[string]int{}
	for _, rg := range rb.Groups {
		for _, r := range groups[rg.Name] {
			if r.Status == StatusRunning && slices.Contains(d.Clubs, r.Club) && !slices.Contains(d.Absent, r.Club) {
				out[r.Club]++
			}
		}
	}
	return out
}

// bestCase is the groups as they end in club's best case: its runners
// still out finish first, on times ahead of every real one, and the other
// clubs' runners still out fail.
func bestCase(groups map[string][]indexed, d Division, club string) map[string][]indexed {
	out := make(map[string][]indexed, len(groups))
	ahead := 0
	for g, runners := range groups {
		rows := slices.Clone(runners)
		for i := range rows {
			r := &rows[i]
			if r.Status != StatusRunning || !slices.Contains(d.Clubs, r.Club) {
				continue
			}
			if r.Club == club {
				ahead++
				r.Status, r.Time = StatusOK, ahead
			} else {
				r.Status, r.Time = StatusDNF, 0
			}
		}
		out[g] = rows
	}
	return out
}
