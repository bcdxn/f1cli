package tealogger

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func LogErr(err error) {
	f, logerr := tea.LogToFile("debug.log", "debug")
	if logerr != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		f.WriteString(fmt.Sprintf("error - %s", err.Error()))
	}
	defer f.Close()
}

func Log(things ...string) {
	f, err := tea.LogToFile("debug.log", "debug")
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
