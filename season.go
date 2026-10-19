package divisionsmatch

import "slices"

// Season is a division's standing over the rounds of a season: the
// rulebook ranks the clubs on the match points of all rounds together,
// and the tie-breaks read the matches and groups of all rounds.
type Season struct {
	Name     string
	Rounds   []DivisionResult
	Standing []SeasonStanding
}

// SeasonStanding is one club's line of the season standing: the totals
// over all rounds and what separated it from a club level on match
// points, then the club's line in each round. A round the club was not
// in has a zero line with the club's name.
type SeasonStanding struct {
	ClubStanding

	Rounds []ClubStanding
}

// Season ranks a division over its rounds, in the order they were run.
// A club is the same club in two rounds when it has the same name in
// both, so the caller gives each club one name for the whole season.
// The clubs are in the order of their first round.
func (rb *Rulebook) Season(name string, rounds []DivisionResult) Season {
	out := Season{Name: name, Rounds: rounds}
	var (
		clubs   []string
		matches []Match
		groups  []Group
	)
	for _, r := range rounds {
		for _, s := range r.Standing {
			if !slices.Contains(clubs, s.Club) {
				clubs = append(clubs, s.Club)
			}
		}
		matches = append(matches, r.Matches...)
		groups = append(groups, r.Groups...)
	}
	rows := make([]ClubStanding, 0, len(clubs))
	for _, club := range clubs {
		rows = append(rows, clubLine(club, false, matches))
	}
	for _, total := range rb.order(rows, matches, groups) {
		line := SeasonStanding{ClubStanding: total}
		for _, r := range rounds {
			i := slices.IndexFunc(r.Standing, func(s ClubStanding) bool { return s.Club == total.Club })
			if i < 0 {
				line.Rounds = append(line.Rounds, ClubStanding{Club: total.Club})
				continue
			}
			line.Rounds = append(line.Rounds, r.Standing[i])
		}
		out.Standing = append(out.Standing, line)
	}
	return out
}
