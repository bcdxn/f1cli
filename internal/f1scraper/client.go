package f1scraper

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bcdxn/f1cli/internal/models"
	"github.com/bcdxn/f1cli/internal/tealogger"
	"github.com/go-resty/resty/v2"
)

const (
	baseUrl = "https://www.formula1.com"
)

var (
	eventClasses = []string{
		".js-practice-1",
		".js-practice-2",
		".js-practice-3",
		".js-sprint-qualifying",
		".js-print-race",
		".js-qualifying",
		".js-race",
	}
)

type F1ScraperClient struct {
	client *resty.Client
	l      tealogger.TeaLogger
}

// New creates a new OpenF1 API client
func New(logger tealogger.TeaLogger) *F1ScraperClient {
	client := resty.New().SetBaseURL(baseUrl)
	return &F1ScraperClient{
		client: client,
		l:      logger,
	}
}

// GetSchedule fetches the official formula1.com schedule html and parses it into a data structure
// usable by the `schedule` program
func (f *F1ScraperClient) GetSchedule() *models.Schedule {
	f.l.Debug("GetSchedule")
	body, err := f.fetchSchedule()

	if err != nil {
		f.l.LogErr(err, "error fetching schedule")
		return nil
	}

	schedule, err := f.parseSchedule(body)

	if err != nil {
		f.l.LogErr(err, "error parsing schedule")
		return nil
	}

	return schedule
}

// GetEventSessions fetches the event detail page from formula1.com and parses it into a data
// structure usable by the `schedule` program.
func (f *F1ScraperClient) GetEventSessions(link string) ([]*models.RaceEventSession, error) {
	f.l.Debug("GetEventSessions")
	return f.fetchEventSessions(link)
}

// GetDriversStandings fetches the drivers page from formula1.com and parses it into a data
// structure usable by the `driverstandings` program.
func (f *F1ScraperClient) GetDriversStandings() []*models.DriverStanding {
	f.l.Debug("GetDriversStandings")
	body, err := f.fetchDriversStandings()

	if err != nil {
		f.l.LogErr(err, "error fetching drivers standings")
		return nil
	}

	standings, err := f.parseDriversStandings(body)

	if err != nil {
		f.l.LogErr(err, "error parsing drivers standings")
		return nil
	}

	return standings
}

func (f *F1ScraperClient) fetchSchedule() (io.Reader, error) {
	resp, err := f.client.R().
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8").
		Get("en/racing/2024.html")

	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(resp.Body())
	return body, nil
}

func (f *F1ScraperClient) fetchEventSessions(link string) ([]*models.RaceEventSession, error) {
	resp, err := f.client.R().
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8").
		Get(link)

	if err != nil {
		return []*models.RaceEventSession{}, errors.New("unable to fetch event details")
	}

	body := bytes.NewReader(resp.Body())

	return f.parseSessionDetails(body)
}

func (f *F1ScraperClient) fetchDriversStandings() (io.Reader, error) {
	resp, err := f.client.R().
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8").
		Get("en/drivers")

	if err != nil {
		return nil, errors.New("unable to fetch drivers standings")
	}

	body := bytes.NewReader(resp.Body())
	return body, nil
}

func (f *F1ScraperClient) parseSchedule(body io.Reader) (*models.Schedule, error) {
	doc, err := goquery.NewDocumentFromReader(body)

	if err != nil {
		return nil, errors.New("error parsing schedule response")
	}

	eventList := doc.Find("main .event-list")
	raceCards := eventList.Find(".race-card-wrapper")
	eventDetailLinks := eventList.Find(".event-item-link")

	events := make([]*models.RaceEvent, raceCards.Size())

	raceCards.Each(func(i int, gq *goquery.Selection) {
		events[i] = f.parseEvent(eventDetailLinks, gq)
	})

	for _, event := range events {
		if event.Upcoming {
			event.IsHeroEvent = true
			f.l.Debug("hero event: ", event.Location)
			break
		}
	}

	return &models.Schedule{
		Events: events,
	}, nil
}

func (f *F1ScraperClient) parseSessionDetails(body io.Reader) ([]*models.RaceEventSession, error) {
	doc, err := goquery.NewDocumentFromReader(body)

	if err != nil {
		return []*models.RaceEventSession{}, errors.New("error parsing event details response")
	}

	eventRows := doc.Find(".f1-race-hub--timetable-listings").Children().Filter(".row")

	sessions := make([]*models.RaceEventSession, 0, eventRows.Size())

	eventRows.Each(func(i int, s *goquery.Selection) {
		c, _ := s.Attr("class")
		f.l.Debugf("div %s", c)
	})

	for _, c := range eventClasses {
		selection := eventRows.Filter(c)
		if selection.Size() == 1 {
			sessions = append(sessions, &models.RaceEventSession{
				Name:     f.safeNodeText(selection, ".f1-timetable--title"),
				StartsAt: f.parseSessionTime(selection),
			})
		}
	}

	f.l.Debugf("found %d sessions", len(sessions))

	return sessions, nil
}

func (f F1ScraperClient) parseEvent(eventDetailLinks, raceCard *goquery.Selection) *models.RaceEvent {
	location := f.safeNodeText(raceCard, ".event-place")
	title := f.safeNodeText(raceCard, ".event-title")
	round := f.safeNodeText(raceCard, ".card-title")
	startsAt, endsAt, err := parseEventDates(raceCard)

	if err != nil {
		f.l.LogErr(err)
	}

	f.l.Debug(location, " ", round)

	r := &models.RaceEvent{
		StartsAt:     startsAt,
		EndsAt:       endsAt,
		GmtOffset:    "",
		Location:     location,
		OfficialName: title,
		Round:        round,
		IsHeroEvent:  false,
		Upcoming:     raceCard.Find(".upcoming").Size() > 0,
	}

	aNode := eventDetailLinks.Filter(fmt.Sprintf("a[data-roundtext='%s']", round))
	link, exists := aNode.First().Attr("href")

	if !exists {
		f.l.LogErr(errors.New("could not parse event detail link"))
	}

	f.l.Debug("\t", link)
	f.l.Debug("\t", strconv.FormatBool(r.Upcoming))
	r.EventDetailLink = link

	return r
}

func (f F1ScraperClient) parseDriversStandings(body io.Reader) ([]*models.DriverStanding, error) {
	doc, err := goquery.NewDocumentFromReader(body)

	if err != nil {
		return nil, errors.New("error parsing schedule response")
	}

	driverNodes := doc.Find("#maincontent").Find(".outline-brand-black.group")

	f.l.Debugf("found %d drivers", driverNodes.Size())

	driversStandings := make([]*models.DriverStanding, driverNodes.Size())

	driverNodes.Each(func(i int, driverNode *goquery.Selection) {
		pos := f.safeNodeText(driverNode, ".f1-heading-black")
		points := f.safeNodeText(driverNode, ".f1-heading-wide")
		name := make([]string, 2)

		driverNode.Find(".f1-driver-name .f1-heading").Each(func(j int, nameNode *goquery.Selection) {
			name[j] = nameNode.Text()
		})

		constructor := f.safeNodeText(driverNode, ".f1-heading.normal-case")

		fullName := strings.Join(name, " ")
		f.l.Debugf("%s %s", pos, fullName)

		driversStandings[i] = &models.DriverStanding{
			Pos:         pos,
			Points:      points,
			Constructor: constructor,
			Name:        fullName,
		}
	})

	return driversStandings, nil
}

func (f *F1ScraperClient) safeNodeText(gq *goquery.Selection, selector string) string {
	node := gq.Find(selector).First()
	if node != nil {
		return strings.Trim(node.Text(), " ")
	}

	f.l.LogErr(fmt.Errorf("failed to parse %s node", selector))
	return ""
}

func parseEventDates(gq *goquery.Selection) (time.Time, time.Time, error) {
	year := "2024"
	monthNode := gq.Find(".month-wrapper")
	startsNode := gq.Find(".start-date")
	endsNode := gq.Find(".end-date")

	if monthNode == nil {
		return time.Time{}, time.Time{}, errors.New("failed to parse .month-wrapper node")
	}

	if startsNode == nil {
		return time.Time{}, time.Time{}, errors.New("failed to parse .start-date node")
	}

	if endsNode == nil {
		return time.Time{}, time.Time{}, errors.New("failed to parse .end-date node")
	}

	month := monthNode.Text()
	startsDate := startsNode.Text()
	endsDate := endsNode.Text()

	months := strings.Split(month, "-")
	startsMonth := months[0]
	endsMonth := months[0]
	if len(months) > 1 {
		endsMonth = months[1]
	}

	startsAt, err := time.Parse("2006-Jan-2", fmt.Sprintf("%s-%s-%s", year, startsMonth, startsDate))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	endsAt, err := time.Parse("2006-Jan-2", fmt.Sprintf("%s-%s-%s", year, endsMonth, endsDate))

	return startsAt, endsAt, err
}

func (f *F1ScraperClient) parseSessionTime(gq *goquery.Selection) time.Time {
	var startTime time.Time
	start, sExists := gq.Attr("data-start-time")
	offset, oExists := gq.Attr("data-gmt-offset")

	f.l.Debug("gmt offset", offset)

	if !sExists {
		f.l.LogErr(errors.New("could not parse session start time"))
		return startTime
	}

	if !oExists {
		f.l.LogErr(errors.New("could not parse session gmt offset"))
		return startTime
	}

	t, err := time.Parse("2006-01-02T15:04:05 -07:00", fmt.Sprintf("%s %s", start, offset))

	if err != nil {
		f.l.LogErr(fmt.Errorf("invalid start time format - %s", start))
		return startTime
	}

	return t.UTC()
}
