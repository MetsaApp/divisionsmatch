package divisionsmatch

// Status is a runner's result status. The values are the MeOS status
// codes, so a caller that holds MeOS codes can convert with a cast.
type Status int

// The statuses. StatusRunning is every runner without a final result: not
// started, out on the course, or with a provisional time.
const (
	StatusRunning          Status = 0
	StatusOK               Status = 1
	StatusNoTiming         Status = 2
	StatusMP               Status = 3
	StatusDNF              Status = 4
	StatusDSQ              Status = 5
	StatusOT               Status = 6
	StatusOutOfCompetition Status = 15
	StatusDNS              Status = 20
	StatusCancelled        Status = 21
	StatusNotCompeting     Status = 99
)

// String is the status as a short code, for messages and tests.
func (s Status) String() string {
	switch s {
	case StatusRunning:
		return "running"
	case StatusOK:
		return "ok"
	case StatusNoTiming:
		return "noTiming"
	case StatusMP:
		return "mp"
	case StatusDNF:
		return "dnf"
	case StatusDSQ:
		return "dsq"
	case StatusOT:
		return "ot"
	case StatusOutOfCompetition:
		return "outOfCompetition"
	case StatusDNS:
		return "dns"
	case StatusCancelled:
		return "cancelled"
	case StatusNotCompeting:
		return "notCompeting"
	}
	return "unknown"
}
