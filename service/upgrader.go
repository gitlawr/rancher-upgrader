package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rancher/go-rancher/catalog"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
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

func UpgradeStack(apiClient *client.RancherClient, config *model.StackUpgrade) error {
	stackName := config.StackName
	var toUpgradeStack *client.Stack
	stacks, err := apiClient.Stack.List(&client.ListOpts{})
	if err != nil {
		log.Errorf("Error %v in listing stacks", err)
		return err
	}
	for _, stack := range stacks.Data {
		if stack.Name == stackName {
			toUpgradeStack = &stack
			break
		}
	}
	if toUpgradeStack == nil {
		log.Errorf("Stack %v is not found.", stackName)
		return err
	}

	if config.ToLatestCatalog {
		if toUpgradeStack.ExternalId == "" {
			log.Error("stack is not deployed from catalog")
			return errors.New("stack is not deployed from catalog")
		}
		log.Infoln("refreshing catalog templates...")
		if err = refreshCatalog(apiClient, config); err != nil {
			return err
		}
		if config.ExternalId == "" {
			latestExtId, err := getTemplateLatestVersion(config, toUpgradeStack.ExternalId)
			if err != nil {
				return err
			}
			config.ExternalId = latestExtId
			template, err := getTemplateVersion(config, latestExtId)
			if err != nil {
				return err
			}
			for k, v := range template.Files {
				if strings.HasPrefix(k, "docker-compose") && config.DockerCompose == "" {
					config.DockerCompose = v.(string)
				} else if strings.HasPrefix(k, "rancher-compose") && config.RancherCompose == "" {
					config.RancherCompose = v.(string)
				}
			}
			if config.Environment == nil && toUpgradeStack.Environment != nil {
				log.Infoln("using previous environment.")
				config.Environment = toUpgradeStack.Environment
			}
		}

		if config.ExternalId == toUpgradeStack.ExternalId {
			log.Infoln("Latest template version already...")
			return nil
		}
	}

	stackUpgrade := &client.StackUpgrade{
		DockerCompose:  config.DockerCompose,
		RancherCompose: config.RancherCompose,
		ExternalId:     config.ExternalId,
		Environment:    config.Environment,
	}
	stack, err := apiClient.Stack.ActionUpgrade(toUpgradeStack, stackUpgrade)
	/*
		serviceIds := stack.ServiceIds

		for _, id := range serviceIds {
			service, err := apiClient.Service.ById(id)
			if err != nil {
				log.Fatalf("Error %v in upgrading stacks", err)
				return err
			}
			if err := wait(apiClient, service); err != nil {
				log.Fatal(err)
				return err
			}
		}
		if err := apiClient.Reload(&stack.Resource, stack); err != nil {
			return err
		}
	*/
	if err := waitStack(apiClient, stack); err != nil {
		log.Error(err.Error())
		return err
	}

	if stack.State != "upgraded" {
		return errors.New("upgrade stack failed.")
	}

	_, err = apiClient.Stack.ActionFinishupgrade(stack)
	if err != nil {
		log.Errorf("Error %v in finishUpgrade of stack %s", err, stack.Name)
		return err
	}
	log.Infof("upgrade stack '%s' success", stack.Name)
	return nil
}

func getProjId(config *model.StackUpgrade) (string, error) {

	client := &http.Client{}

	requestURL := config.CattleUrl

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		log.Infoln("Cannot connect to the rancher server. Please check the rancher server URL")
		return "", err
	}
	req.SetBasicAuth(config.AccessKey, config.SecretKey)
	resp, err := client.Do(req)
	if err != nil {
		log.Infoln("Cannot connect to the rancher server. Please check the rancher server URL")
		return "", err
	}
	defer resp.Body.Close()
	userid := resp.Header.Get("X-Api-User-Id")
	if userid == "" {
		log.Infoln("Cannot get userid")
		err := errors.New("Forbidden")
		return "Forbidden", err

	}
	return userid, nil
}

func getTemplateVersion(config *model.StackUpgrade, externalId string) (*catalog.TemplateVersion, error) {
	u, err := url.Parse(config.CattleUrl)
	if err != nil {
		return &catalog.TemplateVersion{}, err
	}
	projId, err := getProjId(config)
	if err != nil {
		return &catalog.TemplateVersion{}, err
	}
	trimExternalId := externalId[strings.LastIndex(externalId, "/")+1:]
	templateUrl := fmt.Sprintf("%s://%s/v1-catalog/templates/%s?projectId=%s", u.Scheme, u.Host, trimExternalId, projId)

	client := &http.Client{}

	req, err := http.NewRequest("GET", templateUrl, nil)
	if err != nil {
		log.Infoln("Cannot connect to the rancher server. Please check the rancher server URL")
		return &catalog.TemplateVersion{}, err
	}
	req.SetBasicAuth(config.AccessKey, config.SecretKey)
	resp, err := client.Do(req)
	if err != nil {
		log.Infoln("Cannot connect to the rancher server. Please check the rancher server URL")
		return &catalog.TemplateVersion{}, err
	}
	defer resp.Body.Close()

	byteContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &catalog.TemplateVersion{}, err
	}
	tempObj := &catalog.TemplateVersion{}
	if err := json.Unmarshal(byteContent, tempObj); err != nil {
		return &catalog.TemplateVersion{}, errors.Wrap(err, fmt.Sprintf("getTemplateLatestVersion error, Failed to parse: %s", byteContent))
	}
	return tempObj, nil
}

func getTemplateLatestVersion(config *model.StackUpgrade, externalId string) (string, error) {

	tempObj, err := getTemplateVersion(config, externalId)
	if err != nil {
		return "", err
	}
	//no upgrade version
	if tempObj.UpgradeVersionLinks == nil || len(tempObj.UpgradeVersionLinks) == 0 {
		return externalId, nil
	}
	retV := ""
	retRev := 0
	for _, v := range tempObj.UpgradeVersionLinks {
		extId := v.(string)[strings.LastIndex(v.(string), "/")+1:]
		_, _, _, Rev, _ := TemplateURLPath(extId)
		RevI, err := strconv.Atoi(Rev)
		if err != nil {
			return "", err
		}
		if RevI > retRev {
			retRev = RevI
			retV = extId
		}
	}
	return "catalog://" + retV, nil
}

func refreshCatalog(apiClient *client.RancherClient, config *model.StackUpgrade) error {
	u, err := url.Parse(config.CattleUrl)
	if err != nil {
		return err
	}
	projId, err := getProjId(config)
	if err != nil {
		return err
	}
	refreshUrl := fmt.Sprintf("%s://%s/v1-catalog/templates?action=refresh&projectId=%s", u.Scheme, u.Host, projId)

	if err := apiClient.Post(refreshUrl, nil, nil); err != nil {
		return err
	}
	return nil
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

func waitStack(apiClient *client.RancherClient, stack *client.Stack) error {
	for i := 0; i < 36; i++ {
		if err := apiClient.Reload(&stack.Resource, stack); err != nil {
			return err
		}
		if stack.Transitioning != "yes" {
			break
		}
		time.Sleep(5 * time.Second)
	}

	switch stack.Transitioning {
	case "yes":
		return fmt.Errorf("Timeout waiting for %s to finish", stack.Id)
	case "no":
		return nil
	default:
		return fmt.Errorf("Waiting for %s failed: %s", stack.Id, stack.TransitioningMessage)
	}
}

func TemplateURLPath(path string) (string, string, string, string, bool) {
	pathSplit := strings.Split(path, ":")
	switch len(pathSplit) {
	case 2:
		catalog := pathSplit[0]
		template := pathSplit[1]
		templateSplit := strings.Split(template, "*")
		templateBase := ""
		switch len(templateSplit) {
		case 1:
			template = templateSplit[0]
		case 2:
			templateBase = templateSplit[0]
			template = templateSplit[1]
		default:
			return "", "", "", "", false
		}
		return catalog, template, templateBase, "", true
	case 3:
		catalog := pathSplit[0]
		template := pathSplit[1]
		revisionOrVersion := pathSplit[2]
		templateSplit := strings.Split(template, "*")
		templateBase := ""
		switch len(templateSplit) {
		case 1:
			template = templateSplit[0]
		case 2:
			templateBase = templateSplit[0]
			template = templateSplit[1]
		default:
			return "", "", "", "", false
		}
		return catalog, template, templateBase, revisionOrVersion, true
	default:
		return "", "", "", "", false
	}
}
