package divisionsmatch

import (
	"cmp"
	"slices"
)

// The ranking below is the part of a result list the divisionsmatch needs:
// who is placed, on which time, and in which order. A row's section comes
// first, then the order inside it.

// section is the part of a result list a row belongs to.
type section int

const (
	sectionFinished section = iota // OK by time, then out of competition, then no timing.
	sectionRunning                 // No final result yet, by name.
	sectionFailed                  // MP, DNF, DSQ, OT, then name.
	sectionAbsent                  // DNS, cancelled, not competing, then name.
)

// Sub-orders of the finished section.
const (
	rankFirst = iota
	rankMiddle
	rankLast
)

// ranked is a runner placed in the division's result list of one group.
type ranked struct {
	Runner

	index   int
	section section
	place   int
	time    int
}

// indexed is a runner with its position in the caller's slice, which the
// result carries back so the caller can find its own row.
type indexed struct {
	Runner

	index int
}

// rank orders runners as a result list does and places the finished ones:
// equal times share a place and the next place skips accordingly.
func rank(runners []indexed) []ranked {
	rows := make([]ranked, 0, len(runners))
	for _, r := range runners {
		row := ranked{Runner: r.Runner, index: r.index, section: sectionOf(r.Status)}
		if r.Status == StatusOK || r.Status == StatusOutOfCompetition {
			row.time = max(r.Time, 0)
		}
		rows = append(rows, row)
	}
	slices.SortStableFunc(rows, func(a, b ranked) int {
		if a.section != b.section {
			return cmp.Compare(a.section, b.section)
		}
		switch a.section {
		case sectionFinished:
			return cmp.Or(cmp.Compare(finishedRank(a), finishedRank(b)), cmp.Compare(a.time, b.time),
				cmp.Compare(a.Name, b.Name))
		case sectionFailed, sectionAbsent:
			return cmp.Or(cmp.Compare(a.Status, b.Status), cmp.Compare(a.Name, b.Name))
		case sectionRunning:
		}
		return cmp.Compare(a.Name, b.Name)
	})
	prev, place := -1, 0
	for i := range rows {
		r := &rows[i]
		if r.section != sectionFinished || finishedRank(*r) != rankFirst {
			continue
		}
		if r.time != prev {
			place = i + 1
			prev = r.time
		}
		r.place = place
	}
	return rows
}

func sectionOf(s Status) section {
	switch s {
	case StatusOK, StatusNoTiming, StatusOutOfCompetition:
		return sectionFinished
	case StatusMP, StatusDNF, StatusDSQ, StatusOT:
		return sectionFailed
	case StatusDNS, StatusCancelled, StatusNotCompeting:
		return sectionAbsent
	case StatusRunning:
	}
	return sectionRunning
}

// finishedRank orders the finished section: placed rows by time, then
// out of competition rows with a time, then rows without one.
func finishedRank(r ranked) int {
	switch {
	case r.time == 0:
		return rankLast
	case r.Status == StatusOutOfCompetition:
		return rankMiddle
	default:
		return rankFirst
	}
}
