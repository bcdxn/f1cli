package tealogger

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Logger struct {
	name      string
	debug     bool
	debugFile string
	errorFile string
}

func New(name string, opts ...TeaLoggerOption) Logger {
	n := time.Now().Format(time.RFC3339)
	tl := Logger{
		name:      name,
		debug:     false,
		debugFile: "./debug.log",
		errorFile: fmt.Sprintf("./error-%s.log", n),
	}

	for _, opt := range opts {
		opt(&tl)
	}

	tl.Debug("===============================================================================")
	tl.Debug(fmt.Sprintf("--- %s", n))
	tl.Debug("-------------------------------------------------------------------------------")

	return tl
}

type TeaLoggerOption func(l *Logger)

func WithDebugOn() TeaLoggerOption {
	return func(l *Logger) {
		l.debug = true
	}
}

func WithDebugFile(fileName string) TeaLoggerOption {
	return func(l *Logger) {
		l.debugFile = fileName
	}
}

func WithErrorFile(fileName string) TeaLoggerOption {
	return func(l *Logger) {
		l.errorFile = fileName
	}
}

func (t Logger) Error(msg string, things ...any) {
	f, logerr := tea.LogToFile(t.errorFile, "error")
	if logerr != nil {
		fmt.Println(logerr)
		os.Exit(1)
	} else {
		line := make([]string, 1+len(things))
		line[0] = fmt.Sprintf("[%s] %s", t.name, msg)
		for i, thing := range things {
			line[i+1] = fmt.Sprintf("%v", thing)
		}
		f.WriteString(strings.Join(line, " "))
		f.WriteString("\n")
	}
	defer f.Close()
}

func (t Logger) Debug(msg string, things ...any) {
	if t.debug {
		f, err := tea.LogToFile(t.debugFile, "debug")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			line := make([]string, 1+len(things))
			line[0] = fmt.Sprintf("[%s] %s", t.name, msg)
			for i, thing := range things {
				line[i+1] = fmt.Sprintf("%v", thing)
			}
			f.WriteString(strings.Join(line, " "))
			f.WriteString("\n")
		}
		defer f.Close()
	}
}

func (t Logger) Debugf(layout string, things ...any) {
	s := fmt.Sprintf(layout, things...)
	t.Debug(s)
}
