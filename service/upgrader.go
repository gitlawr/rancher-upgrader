package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-upgrader/model"
)

var regTag = regexp.MustCompile(`^[\w]+[\w.-]*`)

func UpgradeServices(apiClient *client.RancherClient, config *model.ServiceUpgrade, pushedImage string) {
	var key, value string
	var secondaryPresent, primaryPresent bool
	serviceSelector := make(map[string]string)

	for key, value = range config.ServiceSelector {
		serviceSelector[key] = value
	}
	batchSize := config.BatchSize
	intervalMillis := config.IntervalMillis
	startFirst := config.StartFirst
	services, err := apiClient.Service.List(&client.ListOpts{})
	if err != nil {
		log.Fatalf("Error %v in listing services", err)
		return
	}

	for _, service := range services.Data {
		secondaryPresent = false
		primaryPresent = false
		primaryLabels := service.LaunchConfig.Labels
		secConfigs := []client.SecondaryLaunchConfig{}
		for _, secLaunchConfig := range service.SecondaryLaunchConfigs {
			labels := secLaunchConfig.Labels
			for k, v := range labels {
				if !strings.EqualFold(k, key) {
					continue
				}
				if !strings.EqualFold(v.(string), value) {
					continue
				}

				secLaunchConfig.ImageUuid = "docker:" + pushedImage
				secLaunchConfig.Labels["io.rancher.container.pull_image"] = "always"
				secConfigs = append(secConfigs, secLaunchConfig)
				secondaryPresent = true
			}
		}

		newLaunchConfig := service.LaunchConfig
		for k, v := range primaryLabels {
			if strings.EqualFold(k, key) {
				if strings.EqualFold(v.(string), value) {
					primaryPresent = true
					newLaunchConfig.ImageUuid = "docker:" + pushedImage
					newLaunchConfig.Labels["io.rancher.container.pull_image"] = "always"
				}
			}
		}

		if !primaryPresent && !secondaryPresent {
			continue
		}

		func(service client.Service, apiClient *client.RancherClient, newLaunchConfig *client.LaunchConfig,
			secConfigs []client.SecondaryLaunchConfig, primaryPresent bool, secondaryPresent bool) {
			upgStrategy := &client.InServiceUpgradeStrategy{
				BatchSize:      batchSize,
				IntervalMillis: intervalMillis * 1000,
				StartFirst:     startFirst,
			}
			if primaryPresent && secondaryPresent {
				upgStrategy.LaunchConfig = newLaunchConfig
				upgStrategy.SecondaryLaunchConfigs = secConfigs
			} else if primaryPresent && !secondaryPresent {
				upgStrategy.LaunchConfig = newLaunchConfig
			} else if !primaryPresent && secondaryPresent {
				upgStrategy.SecondaryLaunchConfigs = secConfigs
			}

			upgradedService, err := apiClient.Service.ActionUpgrade(&service, &client.ServiceUpgrade{
				InServiceStrategy: upgStrategy,
			})
			if err != nil {
				log.Fatalf("Error %v in upgrading service %s", err, service.Id)
				return
			}

			if err := wait(apiClient, upgradedService); err != nil {
				log.Fatal(err)
				return
			}

			if upgradedService.State != "upgraded" {
				return
			}

			_, err = apiClient.Service.ActionFinishupgrade(upgradedService)
			if err != nil {
				log.Fatalf("Error %v in finishUpgrade of service %s", err, upgradedService.Id)
				return
			}
			log.Infof("upgrade service '%s' success", upgradedService.Name)
		}(service, apiClient, newLaunchConfig, secConfigs, primaryPresent, secondaryPresent)
	}
}

func wait(apiClient *client.RancherClient, service *client.Service) error {
	for i := 0; i < 36; i++ {
		if err := apiClient.Reload(&service.Resource, service); err != nil {
			return err
		}
		if service.Transitioning != "yes" {
			break
		}
		time.Sleep(5 * time.Second)
	}

	switch service.Transitioning {
	case "yes":
		return fmt.Errorf("Timeout waiting for %s to finish", service.Id)
	case "no":
		return nil
	default:
		return fmt.Errorf("Waiting for %s failed: %s", service.Id, service.TransitioningMessage)
	}
}

// IsValidTag checks if tag valid as per Docker tag convention
func IsValidTag(tag string) error {
	match := regTag.FindAllString(tag, -1)
	if len(match) == 0 || len(match[0]) > 128 || (len(match[0]) != len(tag)) {
		return fmt.Errorf("Invalid tag %s, tag length must be < 128, must contain [a-zA-Z0-9.-_] characters only, cannot start with '.' or '-'", tag)
	}
	return nil
}
