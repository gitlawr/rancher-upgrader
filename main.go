package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rancher-upgrader/cmd"
	"github.com/urfave/cli"
)

var VERSION = "dev"

func main() {

	app := cli.NewApp()

	app.Name = "rancher-upgrader"
	app.Usage = "Tool for upgrading rancher services"
	app.Before = func(ctx *cli.Context) error {
		if ctx.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Version = VERSION
	app.Author = "Rancher Labs, Inc."
	app.Email = ""
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Debug logging",
		},
		cli.StringFlag{
			Name:   "url",
			Usage:  "Specify the Rancher API endpoint URL",
			EnvVar: "CATTLE_URL",
		},
		cli.StringFlag{
			Name:   "access-key",
			Usage:  "Specify Rancher API access key",
			EnvVar: "CATTLE_ACCESS_KEY",
		},
		cli.StringFlag{
			Name:   "secret-key",
			Usage:  "Specify Rancher API secret key",
			EnvVar: "CATTLE_SECRET_KEY",
		},
	}

	app.Commands = []cli.Command{
		cmd.ServiceCommand(),
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
