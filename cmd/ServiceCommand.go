package cmd

import (
	"errors"
	"strings"

	"github.com/rancher/rancher-upgrader/model"
	"github.com/rancher/rancher-upgrader/service"
	"github.com/urfave/cli"
)

func ServiceCommand() cli.Command {
	serviceFlags := []cli.Flag{
		cli.StringFlag{
			Name:   "env",
			Usage:  "Environment ID",
			EnvVar: "ENVIRONMENT_ID",
		},
		cli.StringSliceFlag{
			Name:  "selector",
			Usage: "service selector labels",
		},
		cli.IntFlag{
			Name:  "batchsize",
			Usage: "batch size",
		},
		cli.IntFlag{
			Name:  "interval",
			Usage: "batch interval in seconds",
		},
		cli.BoolFlag{
			Name:  "startfirst",
			Usage: "start before stopping",
		},
	}

	return cli.Command{
		Name:   "service",
		Usage:  "upgrade services",
		Action: upgrade,
		Flags:  serviceFlags,
	}
}

func upgrade(ctx *cli.Context) error {
	factory := ClientFactory{}
	apiClient, _ := factory.GetClient(ctx)
	selectors := ctx.StringSlice("selector")
	svcSelectors, err := envVarstoMap(selectors)
	if err != nil {
		return err
	}
	config := &model.ServiceUpgrade{
		ServiceSelector: svcSelectors,
		BatchSize:       ctx.Int64("batchsize"),
		IntervalMillis:  ctx.Int64("interval"),
		StartFirst:      ctx.Bool("startfirst"),
	}
	service.UpgradeServices(apiClient, config, "nginx:latest")
	return nil
}

func envVarstoMap(vars []string) (map[string]string, error) {
	m := make(map[string]string)
	for _, s := range vars {
		splits := strings.Split(s, "=")
		if len(splits) != 2 {
			return nil, errors.New("Parse selector '" + s + "' fail, needs the form 'FOO=BAR'")
		}
		key := splits[0]
		val := splits[1]
		m[key] = val
	}
	return m, nil
}
