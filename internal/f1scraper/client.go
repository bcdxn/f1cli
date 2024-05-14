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
}

// New creates a new OpenF1 API client
func New() *F1ScraperClient {
	client := resty.New().SetBaseURL(baseUrl)
	return &F1ScraperClient{
		client: client,
	}
}

func (f *F1ScraperClient) GetSchedule() (*models.Schedule, error) {
	return f.fetchSchedule()
}

func (f *F1ScraperClient) GetEventSessions(link string) ([]*models.RaceEventSession, error) {
	return f.fetchEventSessions(link)
}

func (f *F1ScraperClient) fetchSchedule() (*models.Schedule, error) {
	resp, err := f.client.R().
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8").
		Get("en/racing/2024.html")

	if err != nil {
		return nil, errors.New("unable to fetch schedule")
	}

	body := bytes.NewReader(resp.Body())

	return f.parseSchedule(body)
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
		events[i] = parseEvent(eventDetailLinks, gq)
	})

	for _, event := range events {
		if event.Upcoming {
			event.IsHeroEvent = true
			tealogger.Log("hero event: ", event.Location)
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
		tealogger.Log(fmt.Sprintf("div %s", c))
	})

	for _, c := range eventClasses {
		selection := eventRows.Filter(c)
		if selection.Size() == 1 {
			sessions = append(sessions, &models.RaceEventSession{
				Name:     safeNodeText(selection, ".f1-timetable--title"),
				StartsAt: parseSessionTime(selection),
			})
		}
	}

	tealogger.Log(fmt.Sprintf("found %d sessions", len(sessions)))

	return sessions, nil
}

func parseEvent(eventDetailLinks, raceCard *goquery.Selection) *models.RaceEvent {
	location := safeNodeText(raceCard, ".event-place")
	title := safeNodeText(raceCard, ".event-title")
	round := safeNodeText(raceCard, ".card-title")
	startsAt, endsAt, err := parseEventDates(raceCard)

	if err != nil {
		tealogger.LogErr(err)
	}

	tealogger.Log(location)
	tealogger.Log("\t", round)

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
		tealogger.LogErr(errors.New("could not parse event detail link"))
	}

	tealogger.Log("\t", link)
	tealogger.Log("\t", strconv.FormatBool(r.Upcoming))
	r.EventDetailLink = link

	return r
}

func safeNodeText(gq *goquery.Selection, selector string) string {
	node := gq.Find(selector).First()
	if node != nil {
		return strings.Trim(node.Text(), " ")
	}

	tealogger.LogErr(errors.New(fmt.Sprintf("failed to parse %s node", selector)))
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

func parseSessionTime(gq *goquery.Selection) time.Time {
	var startTime time.Time
	start, sExists := gq.Attr("data-start-time")
	offset, oExists := gq.Attr("data-gmt-offset")

	tealogger.Log("gmt offset", offset)

	if !sExists {
		tealogger.LogErr(errors.New("could not parse session start time"))
		return startTime
	}

	if !oExists {
		tealogger.LogErr(errors.New("could not parse session gmt offset"))
		return startTime
	}

	t, err := time.Parse("2006-01-02T15:04:05 -07:00", fmt.Sprintf("%s %s", start, offset))

	if err != nil {
		tealogger.LogErr(fmt.Errorf("invalid start time format - %s", start))
		return startTime
	}

	return t.UTC()
}
