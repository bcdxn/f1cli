package f1scraper

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

func (f *F1ScraperClient) GetSchedule() ([]models.RaceEvent, error) {
	return f.fetchSchedule()
}

func (f *F1ScraperClient) fetchSchedule() ([]models.RaceEvent, error) {
	resp, err := f.client.R().
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8").
		Get("en/racing/2024.html")

	if err != nil {
		return []models.RaceEvent{}, errors.New("fetch error")
	}

	body := bytes.NewReader(resp.Body())

	return f.parseSchedule(body)
}

func (f *F1ScraperClient) parseSchedule(body io.Reader) ([]models.RaceEvent, error) {
	doc, err := goquery.NewDocumentFromReader(body)

	if err != nil {
		return []models.RaceEvent{}, errors.New("parse error")
	}

	eventList := doc.Find("main .event-list")
	raceCards := eventList.Find(".race-card")

	events := make([]models.RaceEvent, raceCards.Size())

	raceCards.Each(func(i int, gq *goquery.Selection) {
		events[i] = parseEvent(gq)
	})

	return events, nil
}

func parseEvent(gq *goquery.Selection) models.RaceEvent {
	location := safeNodeText(gq, ".event-place")
	title := safeNodeText(gq, ".event-title")
	startsAt, endsAt, err := parseEventDates(gq)

	if err != nil {
		tealogger.LogErr(err)
	}

	return models.RaceEvent{
		StartsAt:     startsAt,
		EndsAt:       endsAt,
		GmtOffset:    "",
		Location:     location,
		OfficialName: title,
	}
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
