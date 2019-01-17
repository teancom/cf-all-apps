package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/cloudfoundry-community/go-cfclient"
	docopt "github.com/docopt/docopt-go"
	yaml "gopkg.in/yaml.v2"
)

type CfConfig struct {
	Api   string `json:"Target"`
	Token string `json:"AccessToken"`
}

type Foundation map[string]Org

type Org map[string]Space

type Space []string

func main() {
	// Add unit testing
	// Add bash/zsh/fish completions
	// Add a homebrew tap to install it
	// Add a pre-compiled version of OSX, linux, and Windows(?)
	// Add ability to use config file vs. "native" CF auth

	usage := `cf-all-apps - List all apps in your targeted CF

	Output defaults to YAML

Usage: 
    cf-all-apps [options]

Options:
    -h, --help              Show this screen
    -j, --json              Output in JSON instead of YAML
    -k, --skip-ssl-verify   Disable cryptographic verification of the CF API
    -r, --running           Only show running apps
    -c, --config            Use a config file for auth instead of relying on the cf cli
`
	arguments, err := docopt.ParseDoc(usage)
	handleErr("Unable to parse arguments.", err)

	version, _ := arguments.Bool("--version")
	jsonOut, _ := arguments.Bool("--json")
	skip, _ := arguments.Bool("--skip-ssl-verify")
	running, _ := arguments.Bool("--running")
	authConfig, _ := arguments.String("--config")

	if version {
		fmt.Println("0.0.1")
		os.Exit(0)
	}

	configFilePath := ""
	if authConfig == "" {
		configFilePath = fmt.Sprintf("%s/.cf/config.json", os.Getenv("HOME"))
	} else {
		configFilePath = authConfig
	}

	configFile, err := os.Open(configFilePath)
	handleErr("Unable to read the specified config file.", err)
	cfConfig, err := readCfConfig(configFile)
	handleErr("Unable to parse your cloudfoundry config file. Try deleting it and re-logging in via the cf cli.", err)

	err = validateCfConfig(cfConfig)
	handleErr("Unable to validate your cloudfoundry config", err)

	c := &cfclient.Config{
		ApiAddress:        cfConfig.Api,
		Token:             strings.TrimPrefix(cfConfig.Token, "bearer "),
		SkipSslValidation: skip,
	}

	client, err := cfclient.NewClient(c)
	handleErr("Unable to contact the CF API", err)

	allApps, err := client.ListApps()
	handleErr("Unable to list the apps", err)

	fmt.Println(outputApps(appsInFoundation(allApps, running), jsonOut))

}

func handleErr(message string, err error) {
	if err != nil {
		fmt.Printf("%s: %s\n", message, err)
		os.Exit(1)
	}
}

func appsInFoundation(apps []cfclient.App, running bool) Foundation {
	foundation := Foundation{}
	for _, app := range apps {
		orgName := app.SpaceData.Entity.OrgData.Entity.Name
		spaceName := app.SpaceData.Entity.Name
		appName := app.Name
		appCount := app.Instances

		if running && app.State != "STARTED" {
			continue // Don't add apps that aren't running when passed -r/--running
		}

		org := foundation[orgName]
		if org == nil {
			// This is the first time we've encountered this org, create it
			org = Org{}
			foundation[orgName] = org
		}

		space := org[spaceName]
		if space == nil {
			// This is the first time we've encountered this space, create it
			space = Space{}
		}
		// Append this app to the org/space, which works even if there are no existing apps
		org[spaceName] = append(space, appName+" "+strconv.Itoa(appCount))
	}

	return foundation
}

func outputApps(foundation Foundation, jsonOut bool) string {
	//	fmt.Printf("%+v\n\n", foundation)
	if jsonOut {
		f, err := json.Marshal(&foundation)
		handleErr("Couldn't convert orgs, spaces, and apps to JSON. Something bad happened.", err)

		return string(f)

	} else {
		f, err := yaml.Marshal(&foundation)
		handleErr("Couldn't convert orgs, spaces, and apps to YAML. Something bad happened.", err)

		return string(f)

	}
}

func readCfConfig(configFile io.Reader) (CfConfig, error) {
	cfConfig := CfConfig{}

	jsonParser := json.NewDecoder(configFile)
	err := jsonParser.Decode(&cfConfig)

	return cfConfig, err
}

func validateCfConfig(cfConfig CfConfig) error {
	if cfConfig.Api == "" {
		return errors.New("You are not currently targeting a CloundFoundry instance")
	}

	if cfConfig.Token == "" {
		return errors.New("You are not currently logged in via the cf api")
	}

	return nil

}
