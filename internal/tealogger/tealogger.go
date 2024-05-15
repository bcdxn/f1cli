package tealogger

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	debugFile = fmt.Sprintf("./debug-%s.log", time.Now())
	errFile   = fmt.Sprintf("./error-%s.log", time.Now())
)

type TeaLogger struct {
	debug     bool
	debugFile string
	errorFile string
}

func New(debug bool) TeaLogger {
	tl := TeaLogger{
		debug:     debug,
		debugFile: fmt.Sprintf("./debug-%s.log", time.Now().Format(time.RFC3339)),
		errorFile: fmt.Sprintf("./error-%s.log", time.Now().Format(time.RFC3339)),
	}

	return tl
}

func (t TeaLogger) LogErr(err error, msgs ...string) {
	f, logerr := tea.LogToFile(errFile, "error")
	if logerr != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		f.WriteString(fmt.Sprintf("%serror - %s\n", strings.Join(msgs, " "), err.Error()))
	}
	defer f.Close()
}

func (t TeaLogger) Debug(things ...string) {
	if t.debug {
		f, err := tea.LogToFile(debugFile, "debug")
		if err != nil {
			fmt.Println(things)
		} else {
			for _, thing := range things {
				f.WriteString(thing + " ")
			}
			f.WriteString("\n")
		}
		defer f.Close()
	}
}

func (t TeaLogger) Debugf(layout string, things ...any) {
	s := fmt.Sprintf(layout, things...)
	t.Debug(s)
}
