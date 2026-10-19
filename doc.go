// Package divisionsmatch computes the Danish divisionsmatch, the club
// league of Dansk Orienterings-Forbund, from a competition's results.
//
// The clubs of one division meet in one competition. Every pair of them
// is a match, and a match is decided by the points their runners take in
// each rulebook group. The calculation is based on divisionsmatchberegning
// by Anders Klinting and uses its words: a division, a match, a group, a
// scale.
//
// The rules are data: one YAML file per rulebook version, embedded in the
// package (see [Rulebooks], [Embedded] and [Latest]) or read from disk
// with [Load]. A [Rulebook] then decides a competition with
// [Rulebook.Compute]:
//
//	rb := divisionsmatch.Latest()
//	result := rb.Compute(divisionsmatch.Config{
//		Divisions: []divisionsmatch.Division{
//			{Name: "1. division", Clubs: []string{"Alpha OK", "Beta OK"}},
//		},
//	}, runners)
package divisionsmatch
