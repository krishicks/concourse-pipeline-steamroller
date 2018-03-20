package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
	steamroller "github.com/krishicks/concourse-pipeline-steamroller"
	yaml "gopkg.in/yaml.v2"
)

type opts struct {
	PipelinePath   FileFlag `long:"pipeline" short:"p" value-name:"PATH" description:"Path to pipeline" required:"true"`
	ConfigPath     FileFlag `long:"config" short:"c" value-name:"PATH" description:"Path to config"`
	ResourceConfig []string `long:"resource-config" value-name:"key=value" description:"resource key/value map"`
}

func main() {
	var o opts
	_, err := flags.Parse(&o)
	if err != nil {
		log.Fatalf("error: %s\n", err)
	}

	var config steamroller.Config
	config.ResourceMap = map[string]string{}

	if o.ConfigPath.Path() != "" {
		var configBytes []byte
		configBytes, err = ioutil.ReadFile(o.ConfigPath.Path())
		if err != nil {
			log.Fatalf("Failed reading config file: %s", err)
		}
		err = yaml.Unmarshal(configBytes, &config)
		if err != nil {
			log.Fatalf("Failed unmarshaling config file: %s", err)
		}
	}

	for _, resourceConfig := range o.ResourceConfig {
		rp := strings.Split(resourceConfig, "=")
		config.ResourceMap[rp[0]] = rp[1]
	}

	pipelineBytes, err := ioutil.ReadFile(o.PipelinePath.Path())
	if err != nil {
		log.Fatalf("failed reading path: %s", err)
	}

	bs, err := steamroller.Steamroll(config.ResourceMap, pipelineBytes)
	if err != nil {
		log.Fatalf("failed steamrolling config: %s", err)
	}

	f := bufio.NewWriter(os.Stdout)

	_, err = f.Write(bs)
	if err != nil {
		log.Fatalf("failed to write steamrolled pipeline to stdout")
	}

	err = f.Flush()
	if err != nil {
		log.Fatalf("failed to flush stdout")
	}
}
