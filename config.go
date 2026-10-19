package divisionsmatch

// Config is a competition's divisionsmatch: the divisions run in it, and
// the organiser's choice of group for the classes where the guess from
// the class name is wrong or missing.
type Config struct {
	Divisions []Division
	// Groups maps a class, by the name the runners carry, to the rulebook
	// group it counts in. An empty group keeps the class out of the match
	// (an open class, a C class). A class not in the map is guessed from
	// its name with [Rulebook.GuessGroup].
	Groups map[string]string
}

// Division is one league division run in the competition: its name, and
// the clubs that meet in it in the order the matches are numbered in.
// Absent are the clubs among them that did not turn up, whose matches
// are decided by the walkover rule rather than by runners.
type Division struct {
	Name   string
	Clubs  []string
	Absent []string
}

// Runner is one row of a result list: who ran, for which club, in which
// class, and how it went.
type Runner struct {
	// Name is only read for a running group: runners entered together
	// carry its name in square brackets, "[g1] Anna Hansen", and only
	// the fastest of them counts.
	Name  string
	Club  string
	Class string
	// Status decides whether the runner can be placed at all.
	Status Status
	// Time is the finish time in whole seconds, as a result list prints
	// it: two runners a tenth apart are level. Zero is no time.
	Time int
}
