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
	}

	app.Commands = []cli.Command{
		cmd.ServiceCommand(),
		cmd.CatalogCommand(),
		cmd.StackCommand(),
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
