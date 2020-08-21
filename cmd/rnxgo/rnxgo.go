// Command-line tool for handling RINEX files - TODO -
package main

import (
	"log"
	"os"

	"github.com/de-bkg/gognss/pkg/rinex"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Usage = "one more RINEX tool"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "starttime, start",
			//Value: "english",
			Usage: "consider epochs beginning at this starttime",
		},
		cli.StringFlag{
			Name:  "endtime, end",
			Usage: "consider epochs up to this endtime",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "diff",
			Usage: "Compare two RINEX files",
			Action: func(c *cli.Context) error {
				if c.NArg() != 2 {
					return cli.NewExitError("diff needs two files to compare", 1)
					//return cli.ShowCommandHelpAndExit(c, "diff", 1)
				}

				fil1 := c.Args().Get(0)
				fil2 := c.Args().Get(1)
				obs1, err := rinex.NewObsFil(fil1)
				if err != nil {
					log.Fatal(err)
				}

				obs2, err := rinex.NewObsFil(fil2)
				if err != nil {
					log.Fatal(err)
				}

				//obs1.Opts.SatSys = []rune("GR")
				err = obs1.Diff(obs2)
				return nil
			},
		},
	}

	// Global options:
	// starttime, endtime
	// satsys

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
