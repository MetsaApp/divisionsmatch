# divisionsmatch

A Go module that calculates the result of a Danish divisionsmatch from the
results of an orienteering competition.

Dansk Orienterings-Forbund (DO-F) sets the rules of the divisionsmatch.
This module implements the 2025 rules of DO-F. It does not change them.
The rules are in a YAML file. You can read them without knowledge of Go.

## What is a divisionsmatch

The divisionsmatch is the Danish club league in orienteering. The clubs of
one division meet at one competition. Each pair of clubs is one match. A
division with 4 clubs has 6 matches.

The runners of the two clubs in a match get points in each group. The
points of all groups added together are the score of the club in that
match. The club with the higher score wins the match and gets match points.
The standing shows the clubs in order of their match points.

## Glossary

This document uses each of these words with one meaning only.

| Word | Meaning |
|------|---------|
| division | The clubs that meet at one competition, for example "3-4. division". |
| match | One pair of clubs in a division. The matches have numbers: m1, m2, and so on. |
| group | A class in the rules, for example H21 or D50. A class in the competition counts in one group, or in no group. |
| scale | The list of points for one group, for example `[8, 7, 6, 5, 4, 3, 2, 1]` for H21. The fastest scoring runner gets the first number. |
| score | The points that a club gets in one match, from all groups added together. |
| match points | The points that a club gets for a match: 2 for a win, 1 for a draw, 0 for a loss. |
| walkover | A match where one club did not come to the competition. |
| running group | Two or more runners who run together. Only the fastest of them can score. The names of these runners contain the name of the running group in square brackets, for example `[g1] Anna`. |
| club limit | The highest number of runners from one club that can score in one group. |
| tie-break | A rule that puts clubs with the same match points in order. |

## How the calculation works

For each match and each group, the program does these steps:

1. It takes the runners of the two clubs in the group, in finish order.
2. It removes runners without a valid result, for example a mispunch.
3. It removes the second and later runners of each running group.
4. It gives the first number of the scale to the first runner, the second number to the next runner, and so on.
5. It skips a runner when that runner's club has the club limit of runners already. The skipped runner does not use a number of the scale.
6. It shares the points equally between runners with the same time. Times are in whole seconds.

The score of a club in the match is the total of the points of its runners
in all groups. Then the program gives the match points.

If one club did not come, the match is a walkover. The club that came gets
the match points for a win. Its score is half of its own runners' points.

The standing puts the clubs in order of match points. For clubs with the
same match points, the program uses the tie-breaks in this order:

1. **Mutual score**: the score of each club in the matches between these clubs only.
2. **Best runners**: the number of groups where the club has the best runner among these clubs. If this is equal, the program counts the second best runners, then the third best runners, and so on.

A division has two or more rounds in a season. The season standing adds
the match points and the scores of all rounds. For clubs with the same
match points, the program uses the same tie-breaks. The tie-breaks use the
matches and the groups of all rounds.

## Check the calculation yourself

You can run the tests on your computer. The tests calculate a real race and
compare the result with the official result of that race.

1. Install Go from <https://go.dev/dl/>.
2. Download this repository as a ZIP file from GitHub.
3. Unzip the file.
4. Open a terminal in the unzipped folder.
5. Type `go test -v -run Golden .` and push Enter.
6. Make sure that the last line starts with `ok`.

Each division writes a line like this:

```
4 standing lines, 6 matches, 228 grid cells, 169 runner rows, 4 season lines agree
```

This line tells you that the program and the official result give the
same standing, the same score in each match, the same score in each group
of each match, the same place and points for each runner, and the same
season standing with the same tie-break values.

### The golden races

Each folder in [testdata](testdata) is one race with its official result:

| Folder | Race | Divisions |
|--------|------|-----------|
| `2026-03-22-1-2-division-ost` | 1. and 2. division Øst, round 1 | 1, 2 |
| `2026-05-10-1-3-division-nord` | 1., 2. and 3. division Nord, round 1 | 1, 2, 3 |
| `2026-05-31-1-2-division-ost` | 1. and 2. division Øst, round 2 | 1, 2 |
| `2026-08-30-2-3-division-nord` | 2. and 3. division Nord, round 2 | 2, 3 |
| `2026-09-06-1-division-nord` | 1. division Nord, round 2 | 1 |
| `2026-10-04-dm-hold` | DM Hold, the final round, 6 clubs | DM Hold |
| `2026-10-04-op-ned-ost` | Op/Ned 3-6. division Øst | 3-4, 4-5, 5-6 |

Each folder contains these files:

| File | Content |
|------|---------|
| `results.xml` | The result list of the competition, as IOF XML 3.0. |
| `official/*.txt` | The official result of each division. |
| `race.yml` | The divisions, the clubs in match order, and the standing and match scores, copied by hand from the official result. |

Some official results are PDF files with text. For these races,
`official/*.txt` is the text of the PDF file, from `pdftotext -layout`.
Other official results are scanned PDF files without text. For these
races, `official/*.txt` is a transcription by hand of the first pages: the
season standing, the matches and the score of each group in each match.
These files have no runner rows, and `race.yml` sets `transcribed: true`.

For round 2, `race.yml` names round 1 with `before`. The test then
calculates the season standing of both rounds and compares it with the
standing at the top of the official result.

All runner names in these files are generic names, for example
"Runner 001". A runner has the same generic name in `results.xml` and in
the official files of the same race. The test does not use names. It finds
each runner by club, class and time.

### Add a race

You can add a race that has an official result. You do not need
to write Go.

1. Make a new folder in `testdata`, for example `testdata/2027-04-11-1-division-vest`.
2. Export the result list from MeOS or O-service as IOF XML 3.0.
3. Save it as `results.xml` in the new folder.
4. Replace each runner name with a generic name, for example "Runner 001".
5. Convert each official PDF file to text with `pdftotext -layout`.
6. Replace the runner names in the text files with the same generic names.
7. Save the text files in a folder `official` in the new folder.
8. Copy `race.yml` from a golden race into the new folder.
9. Change the divisions, clubs, standing and match scores to the values of the new race.
10. If the race is round 2 or later, add the earlier rounds with `before`.
11. Run `go test ./...`.

If an official PDF file is a scan without text, write the first pages by
hand in the layout of `pdftotext`. Look at
[testdata/2026-10-04-dm-hold/official](testdata/2026-10-04-dm-hold/official)
for an example. Set `transcribed: true` for that division.

Do not put real runner names in the repository.

## The rulebook

The rules are in [rulebooks/2025.yml](rulebooks/2025.yml). Each file in
the `rulebooks` folder is one version of the rules. The file name is the
version, for example `2025.yml`.

The file [schema/rulebook.schema.json](schema/rulebook.schema.json)
describes the format. Editors such as VS Code with the YAML extension use
it to check the file while you type.

| Key | Meaning | 2025 value |
|-----|---------|------------|
| `version` | The name of the version. It must be the same as the file name. | `"2025"` |
| `name`, `source` | A name, and the basis for the values. For people to read. | |
| `match_points` | The match points for a `win`, a `draw` and a `loss`. | 2, 1, 0 |
| `club_limit` | The share of a scale that one club can score with. With 0.5, one club can score with 4 runners in a scale of 8. | `0.5` |
| `walkover.score_share` | The share of its own score that a club gets in a walkover. | `0.5` |
| `tie_breaks` | The tie-breaks, in order. The values are `mutual_score` and `best_runners`. | both, in that order |
| `class_names.same_group_suffixes` | Class name endings to remove when the program finds the group of a class. With `A`, "H21A" is in group H21. | `[A]` |
| `groups` | The groups, in order. Each group has a `name`, a `scale`, and optional `aliases` (other class names for the group). | 37 groups |

### Data and code

The YAML file contains each rule that is a value: a number, a list or an
order. The Go code contains each rule that is a step of the calculation.

| In the YAML file | In the Go code |
|------------------|----------------|
| The groups, their order and their scales | The finish order and the places |
| Other class names for a group | The comparison of class names without spaces, dashes and case |
| Match points for a win, a draw and a loss | Shared points for runners with the same time |
| The club limit | Times in whole seconds |
| The walkover share | Which result statuses can score |
| The tie-breaks and their order | How each tie-break works |
| | The running group marks in square brackets in the runner name |

A change to a value needs only the YAML file. A change to a step needs a
change to the Go code.

## When the rules change

Only DO-F changes the rules. This module follows the rules that DO-F
publishes. A rulebook file is one version of the rules, and that version
does not change.

When DO-F publishes a new version of the rules, add a new rulebook file. Do
not change the old file. Do these steps:

1. Copy `rulebooks/2025.yml` to a new file, for example `rulebooks/2027.yml`.
2. Change the version in the new file.
3. Change the values to agree with the new rules.
4. Open a pull request.
5. In the pull request, give the section of the DO-F rules for each change.

Change an existing rulebook file only to correct an error: a value that
does not agree with that version of the DO-F rules and so gives a wrong
calculation. Give the section of the DO-F rules in the pull request.

The tests run on each pull request. If a test fails, the change gives a
different result for a known race. The pull request shows which numbers
are different.

## Use the module in Go

```go
import "github.com/MetsaApp/divisionsmatch"

rb := divisionsmatch.Latest() // or divisionsmatch.Embedded("2025"), or divisionsmatch.Load(path)
result := rb.Compute(divisionsmatch.Config{
	Divisions: []divisionsmatch.Division{
		{Name: "1. division", Clubs: []string{"Alpha OK", "Beta OK", "Gamma OK"}},
	},
	// Optional: a class name to a group. An empty group keeps the class out.
	Groups: map[string]string{"Åben 1": ""},
}, runners) // []divisionsmatch.Runner{{Name, Club, Class, Status, Time}}
```

The public API:

| Name | Use |
|------|-----|
| `Rulebooks()`, `Embedded(version)`, `Latest()` | The rulebooks in the module. |
| `Load(path)`, `Parse(yaml)` | A rulebook from a file or from bytes. |
| `(*Rulebook).Compute(Config, []Runner) Result` | The calculation. |
| `(*Rulebook).Season(name, []DivisionResult) Season` | The season standing of one division over its rounds. Give a club the same name in all rounds. |
| `(*Rulebook).GuessGroup(className)` | The group of a class name, or `""`. |
| `(*Rulebook).Group(name)` | A group and its scale. |
| `Runner`, `Status`, `Config`, `Division` | The input. |
| `Result`, `DivisionResult`, `ClubStanding`, `Placement`, `Match`, `Group`, `GroupRow`, `Season`, `SeasonStanding` | The output. |

`Runner.Time` is the finish time in whole seconds. `Status` uses the MeOS
status codes. `GroupRow.Index` is the position of the runner in the slice
that you gave to `Compute`.

The module uses only the Go standard library and `gopkg.in/yaml.v3`.

## Versions

One release tag versions the Go code and the rulebooks together. A change
to a rulebook is a new release.

## Credits

This module is based on
[divisionsmatchberegning](https://github.com/AndersKlinting/divisionsmatchberegning)
by Anders Klinting. Its words, groups and scales are the basis for the
rulebook in this module. DO-F sets the rules.

## Licence

MIT. See [LICENSE](LICENSE).
