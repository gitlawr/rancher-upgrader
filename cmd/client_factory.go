package cmd

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/rancher/go-rancher/v2"
	"github.com/urfave/cli"
)

type RancherClientFactory interface {
	GetClient(projectID string) (*client.RancherClient, error)
}

type ClientFactory struct{}

func (f *ClientFactory) GetClient(ctx *cli.Context) (*client.RancherClient, error) {
	cattleURL := ctx.GlobalString("url")
	projectID := ctx.String("env")
	url := fmt.Sprintf("%s/projects/%s/schemas", cattleURL, projectID)
	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Timeout:   time.Second * 30,
		Url:       url,
		AccessKey: ctx.GlobalString("access-key"),
		SecretKey: ctx.GlobalString("secret-key"),
	})
	if err != nil {
		logrus.Fatal(err)
		return &client.RancherClient{}, fmt.Errorf("Error in creating API client")
	}
	return apiClient, nil
}
