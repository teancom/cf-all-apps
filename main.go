package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	// Vendor dependencies
	// Add unit testing
	// Add bash/zsh/fish completions
	// Add a homebrew tap to install it
	// Add a pre-compiled version of OSX, linux, and Windows(?)
	// Add ability to use config file vs. "native" CF

	usage := `cf-all-apps - List all apps in your targeted CF

	Output defaults to YAML

Usage: 
    cf-all-apps [options]

Options:
    -h, --help              Show this screen
    -j, --json              Output in JSON instead of YAML
    -k, --skip-ssl-verify   Disable cryptographic verification of the CF API
    -r, --running           Only show running apps
`
	arguments, err := docopt.ParseDoc(usage)
	handleErr("Unable to parse arguments.", err)

	version, _ := arguments.Bool("--version")
	jsonOut, _ := arguments.Bool("--json")
	skip, _ := arguments.Bool("--skip-ssl-verify")
	running, _ := arguments.Bool("--running")

	if version {
		fmt.Println("0.0.1")
		os.Exit(0)
	}

	configFile, err := os.Open(fmt.Sprintf("%s/.cf/config.json", os.Getenv("HOME")))
	handleErr("Unable to read the CF config file. Does it exist? You may have to log in via the cf cli.", err)

	cfConfig := readCfConfig(configFile)

	c := &cfclient.Config{
		ApiAddress:        cfConfig.Api,
		Token:             strings.TrimPrefix(cfConfig.Token, "bearer "),
		SkipSslValidation: skip,
	}

	client, err := cfclient.NewClient(c)
	handleErr("Unable to contact the CF API", err)

	allApps, err := client.ListApps()
	handleErr("Unable to list the apps", err)

	outputApps(appsInFoundation(allApps, running), jsonOut)

}

func handleErr(message string, err error) {
	if err != nil {
		fmt.Sprintf("%s: %s\n", message, err)
		os.Exit(1)
	}
}

func appsInFoundation(apps []cfclient.App, running bool) Foundation {
	foundation := Foundation{}
	for _, app := range apps {
		orgName := app.SpaceData.Entity.OrgData.Entity.Name
		spaceName := app.SpaceData.Entity.Name
		appName := app.Name

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
		org[spaceName] = append(space, appName)
	}

	return foundation
}

func outputApps(foundation Foundation, jsonOut bool) {
	if jsonOut {
		f, err := json.Marshal(&foundation)
		handleErr("Couldn't convert orgs, spaces, and apps to JSON. Something bad happened.", err)

		fmt.Println(string(f))

	} else {
		f, err := yaml.Marshal(&foundation)
		handleErr("Couldn't convert orgs, spaces, and apps to YAML. Something bad happened.", err)

		fmt.Println(string(f))

	}
}

func readCfConfig(configFile io.Reader) CfConfig {
	cfConfig := CfConfig{}

	jsonParser := json.NewDecoder(configFile)
	err := jsonParser.Decode(&cfConfig)
	handleErr("Unable to parse your cloudfoundry config file. Try deleting it and re-logging in via the cf cli.", err)

	if cfConfig.Api == "" {
		fmt.Printf("You are not currently targeting a CloundFoundry instance. Fix that.\n")
		os.Exit(1)
	}

	if cfConfig.Token == "" {
		fmt.Printf("You are not currently logged in via the cf api. Fix that.\n")
		os.Exit(1)
	}
	return cfConfig
}
