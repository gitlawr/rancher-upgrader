package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rancher/rancher-upgrader/model"
	"github.com/rancher/rancher-upgrader/service"
	"github.com/urfave/cli"
)

func CatalogCommand() cli.Command {
	catalogFlags := []cli.Flag{
		cli.StringFlag{
			Name:   "envurl",
			Usage:  "Environment ENDPOINT URL",
			EnvVar: "CATTLE_URL",
		},
		cli.StringFlag{
			Name:   "accesskey",
			Usage:  "Environment ACCESS KEY",
			EnvVar: "CATTLE_ACCESS_KEY",
		},
		cli.StringFlag{
			Name:   "secretkey",
			Usage:  "Environment SECRET KEY",
			EnvVar: "CATTLE_SECRET_KEY",
		},
		cli.StringFlag{
			Name:  "repourl",
			Usage: "git url for catalog repo",
		},
		cli.StringFlag{
			Name:  "branch",
			Usage: "catalog repo branch",
		},
		cli.StringFlag{
			Name:  "user",
			Usage: "git username",
		},
		cli.StringFlag{
			Name:  "password",
			Usage: "git password",
		},
		cli.StringFlag{
			Name:  "cacheroot",
			Usage: "cache directory to store catalog items",
		},
		cli.StringFlag{
			Name:  "foldername",
			Usage: "catalog template folder name",
		},
		cli.BoolFlag{
			Name:  "system",
			Usage: "catalog template type",
		},
		cli.StringFlag{
			Name:  "compose-file",
			Usage: "docker-compose file path",
			Value: "./docker-compose.yml",
		},
		cli.StringFlag{
			Name:  "rancher-file",
			Usage: "rancher-compose file path",
			Value: "./rancher-compose.yml",
		},
		cli.StringFlag{
			Name:  "readme",
			Usage: "readme file path",
		},
	}

	return cli.Command{
		Name:   "catalog",
		Usage:  "upgrade catalog",
		Action: upgradeCatalog,
		Flags:  catalogFlags,
	}
}

func upgradeCatalog(ctx *cli.Context) error {
	//factory := ClientFactory{}
	//apiClient, _ := factory.GetClient(ctx)

	composeFile := ctx.String("compose-file")
	rancherFile := ctx.String("rancher-file")
	dockerCompose := ""
	rancherCompose := ""
	if composeFile != "" {
		cdat, err := ioutil.ReadFile(composeFile)
		check(err)
		dockerCompose = string(cdat)
	}
	if rancherFile != "" {
		rdat, err := ioutil.ReadFile(rancherFile)
		check(err)
		rancherCompose = string(rdat)
	}
	config := &model.CatalogUpgrade{
		CacheRoot:          ctx.String("cacheroot"),
		GitUrl:             ctx.String("repourl"),
		GitBranch:          ctx.String("branch"),
		TemplateFolderName: ctx.String("foldername"),
		TemplateIsSystem:   ctx.Bool("system"),
		GitUser:            ctx.String("user"),
		GitPassword:        ctx.String("password"),

		DockerCompose:  dockerCompose,
		RancherCompose: rancherCompose,
	}
	if err := service.UpgradeCatalog(config); err != nil {
		return err
	}
	return nil
}

func check(e error) {
	if e != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", e)
		os.Exit(1)
	}
}
