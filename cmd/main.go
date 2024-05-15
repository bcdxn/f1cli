package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bcdxn/f1cli/internal/driver"
	"github.com/bcdxn/f1cli/internal/schedule"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "f1",
		Usage: "Formula 1 in your terminal",
		Action: func(c *cli.Context) error {
			cli.ShowAppHelp(c)
			return nil
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Log debug statements to file",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "schedule",
				Usage: "View the Formula 1 schedule",
				// Flags: []cli.Flag{
				// 	&cli.BoolFlag{
				// 		Name:  "tracktime",
				// 		Usage: "Show session times in track time (defaults to local time)",
				// 	},
				// },
				Action: func(cCtx *cli.Context) error {
					schedule.RunProgram(schedule.ScheduleOptions{
						DisplayTrackTimes: false,
						Debug:             cCtx.Bool("debug"),
					})
					return nil
				},
			},
			{
				Name:  "results",
				Usage: "View F1 event results",
				Action: func(cCtx *cli.Context) error {
					fmt.Println("coming soon 👀")
					return nil
				},
			},
			{
				Name:  "standings",
				Usage: "View championship standings",
				Subcommands: []*cli.Command{
					{
						Name:  "drivers",
						Usage: "Drivers World Championship standings",
						Action: func(cCtx *cli.Context) error {
							driver.RunProgram(driver.StandingsOptions{
								Debug: cCtx.Bool("debug"),
							})
							return nil
						},
					},
					{
						Name:  "constructors",
						Usage: "Constructors World Championship standings",
						Action: func(cCtx *cli.Context) error {
							fmt.Println("coming soon 👀")
							return nil
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
