package divisionsmatch_test

import (
	"math"
	"strings"
	"testing"

	"github.com/MetsaApp/iofxml"

	dm "github.com/MetsaApp/divisionsmatch"
)

// runnersOf is the runners of an IOF XML 3.0 ResultList, the way MeOS and
// O-service export one: each individual runner's class, club, status and
// finish time. A runner keeps the name the file gives, given and family
// name joined, which carries a running group's "[g1]" through. Relay
// results are skipped, and so is a row the file marks as moved to another
// class.
func runnersOf(l *iofxml.ResultList) []dm.Runner {
	var runners []dm.Runner
	for _, c := range l.ClassResults {
		for _, p := range c.PersonResults {
			r := dm.Runner{Name: p.Person.Name.Full(), Class: strings.TrimSpace(c.Class.Name)}
			if p.Organisation != nil {
				r.Club = strings.TrimSpace(p.Organisation.Name)
			}
			if len(p.Results) > 0 {
				res := p.Results[0]
				if !res.Status.Trim().InClass() {
					continue
				}
				r.Status = statusOf(res.Status.Trim())
				if res.Time != nil && *res.Time > 0 {
					// A result list prints whole seconds; the standard may
					// write decimals.
					r.Time = int(math.Floor(*res.Time))
				}
			}
			runners = append(runners, r)
		}
	}
	return runners
}

// statusOf maps the standard's ResultStatus onto a runner's. Finished is a
// time not yet confirmed, so it is no final result.
func statusOf(s iofxml.ResultStatus) dm.Status {
	switch s {
	case iofxml.StatusOK:
		return dm.StatusOK
	case iofxml.StatusMissingPunch:
		return dm.StatusMP
	case iofxml.StatusDidNotFinish:
		return dm.StatusDNF
	case iofxml.StatusDisqualified:
		return dm.StatusDSQ
	case iofxml.StatusOverTime:
		return dm.StatusOT
	case iofxml.StatusNotCompeting:
		return dm.StatusNotCompeting
	case iofxml.StatusDidNotStart, iofxml.StatusSportingWithdrawal:
		return dm.StatusDNS
	case iofxml.StatusCancelled:
		return dm.StatusCancelled
	case iofxml.StatusFinished, iofxml.StatusActive, iofxml.StatusInactive,
		iofxml.StatusMoved, iofxml.StatusMovedUp, iofxml.StatusDidNotEnter:
		return dm.StatusRunning
	}
	return dm.StatusRunning
}

// A result list gives every runner with what the divisionsmatch reads:
// the time in whole seconds, the status mapped, a moved runner left out,
// a runner with no result still out, and a relay team skipped.
func TestRunnersOf(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<ResultList xmlns="http://www.orienteering.org/datastandard/3.0" iofVersion="3.0">
  <Event><Name>Test</Name></Event>
  <ClassResult>
    <Class><Name>H21-</Name></Class>
    <PersonResult>
      <Person><Name><Family>Hansen</Family><Given>[g1] Anna</Given></Name></Person>
      <Organisation><Name>Alpha OK</Name></Organisation>
      <Result><Time>1800.7</Time><Status>OK</Status></Result>
    </PersonResult>
    <PersonResult>
      <Person><Name><Family>B</Family><Given>Bo</Given></Name></Person>
      <Organisation><Name>Beta OK</Name></Organisation>
      <Result><Status>MissingPunch</Status></Result>
    </PersonResult>
    <PersonResult>
      <Person><Name><Family>C</Family><Given>Cy</Given></Name></Person>
      <Organisation><Name>Beta OK</Name></Organisation>
      <Result><Status>Moved</Status></Result>
    </PersonResult>
    <PersonResult>
      <Person><Name><Family>D</Family><Given>Di</Given></Name></Person>
      <Organisation><Name>Beta OK</Name></Organisation>
    </PersonResult>
  </ClassResult>
  <ClassResult>
    <Class><Name>Stafet</Name></Class>
    <TeamResult><Name>Alpha 1</Name></TeamResult>
  </ClassResult>
</ResultList>`
	read, err := iofxml.Read(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	equal(t, []dm.Runner{
		{Name: "[g1] Anna Hansen", Club: "Alpha OK", Class: "H21-", Status: dm.StatusOK, Time: 1800},
		{Name: "Bo B", Club: "Beta OK", Class: "H21-", Status: dm.StatusMP},
		{Name: "Di D", Club: "Beta OK", Class: "H21-", Status: dm.StatusRunning},
	}, runnersOf(read.ResultList))
}
