package f1livetiming

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bcdxn/f1cli/internal/domain"
)

func TestProcessReferenceMessage(t *testing.T) {

	t.Run("Practice", func(t *testing.T) {
		t.Parallel()
		td := testdataDir()
		ref, _ := os.ReadFile(path.Join(td, "ref-msg-practice.json"))

		c := New(WithLogger(testLogger(t)))
		go c.processMessage(ref)

		wait := 2
		for wait > 0 {
			select {
			case meeting := <-c.Meeting():
				wait--
				if meeting.Session.Type != domain.SessionTypePractice {
					t.Errorf("expected session type '%s' but found '%s'", domain.SessionTypePractice, meeting.Session.Type)
				}
				if meeting.Session.Part != 0 {
					t.Errorf("expected session part %d but found %d", 0, meeting.Session.Part)
				}
				if meeting.Session.CurrentLap != 0 {
					t.Errorf("expected lap count %d but found %d", 0, meeting.Session.CurrentLap)
				}
				if meeting.Session.TotalLaps != 0 {
					t.Errorf("expected lap count %d but found %d", 0, meeting.Session.TotalLaps)
				}
			case drivers := <-c.Drivers():
				wait--
				if len(drivers) != 20 {
					t.Errorf("expected %d drivers but found %d", 20, len(drivers))
				}
				if drivers["1"].Name != "Max Verstappen" {
					t.Errorf("expected name '%s' but found '%s'", "Max Verstappen", drivers["1"].Name)
				}
				if drivers["97"].Name != "Robert Shwartzman" {
					t.Errorf("expected name '%s' but found '%s'", "Robert Shwartzman", drivers["97"].Name)
				}
				if drivers["81"].TimingData.BestLapTime != "1:20.515" {
					t.Errorf("expected best lap time '%s' but found '%s'", "1:20.515", drivers["81"].TimingData.BestLapTime)
				}
				if drivers["81"].TimingData.Position != 9 {
					t.Errorf("expected position %d but found %d", 9, drivers["81"].TimingData.Position)
				}
				if drivers["1"].TimingData.TireCompound != domain.TireCompoundMedium {
					t.Errorf("expected tire compound '%s' but found '%s'", domain.TireCompoundMedium, drivers["1"].TimingData.TireCompound)
				}
				if drivers["1"].TimingData.NumberOfLaps != 6 {
					t.Errorf("expected stint laps %d but found %d", 6, drivers["1"].TimingData.NumberOfLaps)
				}
				// case <-c.RaceCtrlMsgs():
				// 	wait--
			}
		}
	})
	t.Run("Qualifying", func(t *testing.T) {
		t.Parallel()
		td := testdataDir()
		ref, _ := os.ReadFile(path.Join(td, "ref-msg-qualifying.json"))

		c := New(WithLogger(testLogger(t)))
		go c.processMessage(ref)

		wait := 2
		fmt.Println("waiting for channel messages")

		for wait > 0 {
			select {
			case meeting := <-c.Meeting():
				wait--
				if meeting.Session.Type != domain.SessionTypeQualifying {
					t.Errorf("expected session type '%s' but found '%s'", domain.SessionTypeQualifying, meeting.Session.Type)
				}
				if meeting.Session.Part != 1 {
					t.Errorf("expected session part %d but found %d", 1, meeting.Session.Part)
				}
				if meeting.Session.CurrentLap != 0 {
					t.Errorf("expected lap count %d but found %d", 0, meeting.Session.CurrentLap)
				}
				if meeting.Session.TotalLaps != 0 {
					t.Errorf("expected lap count %d but found %d", 0, meeting.Session.TotalLaps)
				}
			case drivers := <-c.Drivers():
				wait--
				if len(drivers) != 20 {
					t.Errorf("expected %d drivers but found %d", 20, len(drivers))
				}
				if drivers["1"].Name != "Max Verstappen" {
					t.Errorf("expected name '%s' but found '%s'", "Max Verstappen", drivers["1"].Name)
				}
				if drivers["24"].Name != "Zhou Guanyu" {
					t.Errorf("expected name '%s' but found '%s'", "Zhou Guanyu", drivers["1"].Name)
				}
				if drivers["81"].TimingData.BestLapTimes[0] != "1:23.640" {
					t.Errorf("expected best lap time '%s' but found '%s'", "1:23.640", drivers["81"].TimingData.BestLapTimes[0])
				}
				if drivers["81"].TimingData.Position != 4 {
					t.Errorf("expected position %d but found %d", 4, drivers["81"].TimingData.Position)
				}
				if drivers["1"].TimingData.TireCompound != domain.TireCompoundSoft {
					t.Errorf("expected tire compound '%s' but found '%s'", domain.TireCompoundSoft, drivers["1"].TimingData.TireCompound)
				}
				if drivers["1"].TimingData.NumberOfLaps != 3 {
					t.Errorf("expected stint laps %d but found %d", 3, drivers["1"].TimingData.NumberOfLaps)
				}
				// case <-c.RaceCtrlMsgs():
				// 	wait--
			}
		}
	})

	t.Run("Race", func(t *testing.T) {
		t.Parallel()
		td := testdataDir()
		ref, _ := os.ReadFile(path.Join(td, "ref-msg-race.json"))

		c := New(WithLogger(testLogger(t)))
		go c.processMessage(ref)

		wait := 2
		fmt.Println("waiting for channel messages")

		for wait > 0 {
			select {
			case meeting := <-c.Meeting():
				wait--
				if meeting.Session.Type != domain.SessionTypeRace {
					t.Errorf("expected session type '%s' but found '%s'", domain.SessionTypeRace, meeting.Session.Type)
				}
				if meeting.Session.Part != 0 {
					t.Errorf("expected session part %d but found %d", 0, meeting.Session.Part)
				}
				if meeting.Session.CurrentLap != 1 {
					t.Errorf("expected lap count %d but found %d", 1, meeting.Session.CurrentLap)
				}
				if meeting.Session.TotalLaps != 58 {
					t.Errorf("expected total laps %d but found %d", 58, meeting.Session.TotalLaps)
				}
				if meeting.Session.Status != domain.SessionStatusPending {
					t.Errorf("expected status '%s' but found '%s'", domain.SessionStatusPending, meeting.Session.Status)
				}
			case drivers := <-c.Drivers():
				wait--
				if len(drivers) != 20 {
					t.Errorf("expected %d drivers but found %d", 20, len(drivers))
				}
				if drivers["1"].Name != "Max Verstappen" {
					t.Errorf("expected name '%s' but found '%s'", "Max Verstappen", drivers["1"].Name)
				}
				if drivers["24"].Name != "Zhou Guanyu" {
					t.Errorf("expected name '%s' but found '%s'", "Zhou Guanyu", drivers["1"].Name)
				}
				if drivers["81"].TimingData.BestLapTime != "" {
					t.Errorf("expected best lap time '%s' but found '%s'", "", drivers["81"].TimingData.BestLapTime)
				}
				if drivers["81"].TimingData.Position != 2 {
					t.Errorf("expected position %d but found %d", 2, drivers["81"].TimingData.Position)
				}
				if drivers["1"].TimingData.TireCompound != domain.TireCompoundUnknown {
					t.Errorf("expected tire compound '%s' but found '%s'", domain.TireCompoundUnknown, drivers["1"].TimingData.TireCompound)
				}
				if drivers["1"].TimingData.NumberOfLaps != 0 {
					t.Errorf("expected stint laps %d but found %d", 0, drivers["1"].TimingData.NumberOfLaps)
				}
				// case <-c.RaceCtrlMsgs():
				// 	wait--
			}
		}
	})
}

func TestProcessChangeMessage(t *testing.T) {

	t.Run("Qualifying", func(t *testing.T) {
		td := testdataDir()
		t.Run("TimingData", func(t *testing.T) {
			c := newReferenecedClient(t, path.Join(td, "ref-msg-qualifying.json"))
			change, _ := os.ReadFile(path.Join(td, "ch-msg-qual-timingdata.json"))
			go c.processMessage(change)

			var drivers map[string]domain.Driver

			wait := true
			for wait {
				select {
				case <-c.Meeting():
				case <-c.RaceCtrlMsgs():
					// we only care about the drivers channel in this test
				case drivers = <-c.Drivers():
					wait = false
				}
			}

			if drivers["81"].TimingData.Position != 8 {
				t.Errorf("expected position %d but found %d", 8, drivers["81"].TimingData.Position)
			}
			if drivers["27"].TimingData.LeaderGap != "+0.420" {
				t.Errorf("expected leader gap '%s' but found '%s'", "+0.420", drivers["27"].TimingData.LeaderGap)
			}
			if drivers["27"].TimingData.IntervalGap != "+0.040" {
				t.Errorf("expected interval gap '%s' but found '%s'", "+0.040", drivers["27"].TimingData.IntervalGap)
			}
		})
	})
	t.Run("Race", func(t *testing.T) {
		td := testdataDir()
		t.Run("TimingData", func(t *testing.T) {
			c := newReferenecedClient(t, path.Join(td, "ref-msg-race.json"))
			change, _ := os.ReadFile(path.Join(td, "ch-msg-race-timingdata.json"))
			go c.processMessage(change)

			var drivers map[string]domain.Driver

			wait := true
			for wait {
				select {
				case <-c.Meeting():
				case <-c.RaceCtrlMsgs():
					// we only care about the drivers channel in this test
				case drivers = <-c.Drivers():
					wait = false
				}
			}

			if drivers["61"].TimingData.Position != 18 {
				t.Errorf("expected position %d but found %d", 18, drivers["61"].TimingData.Position)
			}
			if drivers["23"].TimingData.Position != 16 {
				t.Errorf("expected position %d but found %d", 16, drivers["23"].TimingData.Position)
			}
			if drivers["23"].TimingData.LeaderGap != "+4.625" {
				t.Errorf("expected position %s but found %s", "+4.625", drivers["23"].TimingData.LeaderGap)
			}
			if drivers["23"].TimingData.IntervalGap != "+0.133" {
				t.Errorf("expected position %s but found %s", "+0.133", drivers["23"].TimingData.IntervalGap)
			}
		})

		t.Run("SessionData", func(t *testing.T) {
			c := newReferenecedClient(t, path.Join(td, "ref-msg-race.json"))
			change, _ := os.ReadFile(path.Join(td, "ch-msg-race-sessiondata.json"))
			go c.processMessage(change)

			var meeting domain.Meeting

			wait := true
			for wait {
				select {
				// we only care about the drivers channel in this test
				case meeting = <-c.Meeting():
					wait = false
				case <-c.RaceCtrlMsgs():
				case <-c.Drivers():
				}
			}

			if meeting.Session.Status != domain.SessionStatusStarted {
				t.Errorf("expected status '%s' but found '%s'", domain.SessionStatusStarted, meeting.Session.Status)
			}
		})
	})
}

// getTestdataDir gets the testdata directory path relative to the invocation of the tests.
func testdataDir() string {
	_, p, _, _ := runtime.Caller(0)
	return path.Join(filepath.Dir(p), "testdata")
}

// testLogger creates a new logger to be used in tests that writes all logs to /dev/null so they
// don't uglify the test output.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newReferenecedClient(t *testing.T, refpath string) Client {
	t.Helper()
	ref, _ := os.ReadFile(refpath)
	c := New(WithLogger(testLogger(t)))
	go c.processMessage(ref)

	wait := 2
	for wait > 0 {
		select {
		case <-c.Meeting():
			wait--
		case <-c.Drivers():
			wait--
		}
	}

	return c
}
