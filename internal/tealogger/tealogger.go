package tealogger

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	debugFile = fmt.Sprintf("./debug-%s.log", time.Now())
	errFile   = fmt.Sprintf("./error-%s.log", time.Now())
)

func LogErr(err error) {
	f, logerr := tea.LogToFile(errFile, "debug")
	if logerr != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		f.WriteString(fmt.Sprintf("error - %s", err.Error()))
	}
	defer f.Close()
}

func Log(things ...string) {
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
