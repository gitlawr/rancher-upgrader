package cmd

import (
	"bufio"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rancher-upgrader/model"
	"github.com/rancher/rancher-upgrader/service"
	"github.com/urfave/cli"
)

func StackCommand() cli.Command {
	stackFlags := []cli.Flag{
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
			Name:  "stackname",
			Usage: "stack name to upgrade",
		},
		cli.StringFlag{
			Name:  "env-file",
			Usage: "env file to use in catalog",
		},
		cli.StringFlag{
			Name:  "compose-file",
			Usage: "docker compose file for stack upgrade",
		},
		cli.StringFlag{
			Name:  "rancher-file",
			Usage: "rancher compose file for stack upgrade",
		},
		cli.BoolFlag{
			Name:  "tolatest",
			Usage: "upgrade stack to latest catalog version",
		},
	}

	return cli.Command{
		Name:   "stack",
		Usage:  "upgrade stack",
		Action: upgradeStack,
		Flags:  stackFlags,
	}
}

func upgradeStack(ctx *cli.Context) error {
	factory := ClientFactory{}
	apiClient, _ := factory.GetClient(ctx)

	var envs map[string]interface{}
	if ctx.String("env-file") != "" {
		envs = parseCustomEnvFile(ctx.String("env-file"))
	}
	config := &model.StackUpgrade{
		CattleUrl:       ctx.String("envurl"),
		AccessKey:       ctx.String("accesskey"),
		SecretKey:       ctx.String("secretkey"),
		StackName:       ctx.String("stackname"),
		Environment:     envs,
		ToLatestCatalog: ctx.Bool("tolatest"),
	}
	service.UpgradeStack(apiClient, config)
	return nil
}

func parseCustomEnvFile(file string) map[string]interface{} {
	variables := map[string]interface{}{}

	f, err := os.Open(file)
	if err != nil {
		logrus.Fatal(err)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := scanner.Text()
		parts := strings.SplitN(t, "=", 2)
		if len(parts) == 1 {
			variables[parts[0]] = ""
		} else {
			variables[parts[0]] = parts[1]
		}
	}

	if scanner.Err() != nil {
		logrus.Fatal(scanner.Err())
	}

	return variables
}
