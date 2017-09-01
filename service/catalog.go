package service

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/rancher/rancher-upgrader/git"
	"github.com/rancher/rancher-upgrader/model"
)

/*
func getConfig(apiClient *catalog.RancherClient, externalId string) {
	//templateId := externalId

	template, err := apiClient.Template.ById("")
	if err != nil {
		logrus.Fatalf("get catalog template error:%v\n", err)
	}
	tv, err := apiClient.TemplateVersion.ById("")
	apiClient.
	tv.TemplateId
	catalogName, templateName, templateBase, revisionOrVersion, _ := parse.TemplateURLPath(catalogTemplateVersion)

}
*/

func UpgradeCatalog(config *model.CatalogUpgrade) error {
	/*
		opt := &catalog.ClientOpts{
			Url:       "",
			AccessKey: "",
			SecretKey: "",
		}
		client, _ := catalog.NewRancherClient(opt)
		catalog, _ := client.Catalog.ById("")
		template, _ := client.Template.ById("")
	*/
	repoPath, _, err := prepareGitRepoPath(config)
	if err != nil {
		logrus.Errorf("Prepare Git repo path got error:%v", err)
		return err
	}

	if err := generateNewTemplateVersion(repoPath, config); err != nil {
		return err
	}

	return nil
}

func dirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

//
func prepareGitRepoPath(config *model.CatalogUpgrade) (string, string, error) {
	branch := config.GitBranch
	if config.GitBranch == "" {
		branch = "master"
	}

	sum := md5.Sum([]byte(config.GitUrl + branch))
	repoBranchHash := hex.EncodeToString(sum[:])
	repoPath := path.Join(config.CacheRoot, repoBranchHash)

	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return "", "", errors.Wrap(err, "mkdir failed")
	}

	if err := git.Clone(repoPath, config.GitUrl, branch); err != nil {
		return "", "", errors.Wrap(err, "Clone failed")
	}

	commit, err := git.HeadCommit(repoPath)
	if err != nil {
		err = errors.Wrap(err, "Retrieving head commit failed")
	}
	return repoPath, commit, err
}

func generateNewTemplateVersion(repoPath string, config *model.CatalogUpgrade) error {

	templatePath := ""
	if config.TemplateIsSystem == false {
		templatePath = filepath.Join(repoPath, "templates", config.TemplateFolderName)
	} else {
		templatePath = filepath.Join(repoPath, "infra-templates", config.TemplateFolderName)
	}

	lv, err := GetLatestVersion(templatePath)

	if err != nil {
		logrus.Errorf("get template version error: %v", err)
		return err
	}
	newV := lv + 1

	if err = os.Mkdir(filepath.Join(templatePath, strconv.Itoa(newV)), 0755); err != nil {
		logrus.Errorf("prepare new template version got error: %v", err)
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(templatePath, strconv.Itoa(newV), "docker-compose.yml"), []byte(config.DockerCompose), 0755); err != nil {
		logrus.Errorf("prepare new template version got error: %v", err)
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(templatePath, strconv.Itoa(newV), "rancher-compose.yml"), []byte(config.RancherCompose), 0755); err != nil {
		logrus.Errorf("prepare new template version got error: %v", err)
		return err
	}

	if config.Readme != "" {
		if err = ioutil.WriteFile(filepath.Join(templatePath, strconv.Itoa(newV), "README.md"), []byte(config.Readme), 0755); err != nil {
			logrus.Errorf("prepare new template version got error: %v", err)
			return err
		}
	}

	repoUrl := config.GitUrl
	if strings.HasPrefix(repoUrl, "https") {
		if config.GitUser != "" && config.GitPassword != "" {
			repoUrl = strings.Replace(repoUrl, "https://", "https://"+config.GitUser+":"+config.GitPassword+"@", 1)
		} else {
			logrus.Fatalf("username/password for git repo not provided.\n")
		}
	}

	if err = git.LazyPush(templatePath, repoUrl, config.GitBranch); err != nil {
		logrus.Errorf("prepare new template version got error: %v", err)
		return err
	}

	return nil
}

//GetLatestVersion returns latest version in the catalog template path
func GetLatestVersion(templatePath string) (int, error) {
	latestVersion := -1
	files, err := ioutil.ReadDir(templatePath)
	if err != nil {
		logrus.Errorf("read templatepath fail:%v", err.Error())
		return latestVersion, err
	}
	for _, f := range files {
		i, err := strconv.Atoi(f.Name())
		if err != nil {
			continue
		}
		if i > latestVersion {
			latestVersion = i
		}
	}
	return latestVersion, nil
}
