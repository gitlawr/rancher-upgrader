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
	envEndpoint := ctx.String("envurl")
	//projectID := ctx.String("env")
	//url := fmt.Sprintf("%s/projects/%s/schemas", cattleURL, projectID)
	url := envEndpoint + "/schemas"
	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Timeout:   time.Second * 30,
		Url:       url,
		AccessKey: ctx.String("accesskey"),
		SecretKey: ctx.String("secretkey"),
	})
	if err != nil {
		logrus.Fatal(err)
		return &client.RancherClient{}, fmt.Errorf("Error in creating API client")
	}
	return apiClient, nil
}
