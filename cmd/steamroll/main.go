package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	steamroller "github.com/krishicks/concourse-pipeline-steamroller"
	yaml "gopkg.in/yaml.v2"
)

type opts struct {
	PipelinePath FileFlag `long:"pipeline" short:"p" value-name:"PATH" description:"Path to pipeline"`
	ConfigPath   FileFlag `long:"config" short:"c" value-name:"PATH" description:"Path to config"`
}

func main() {
	var o opts
	_, err := flags.Parse(&o)
	if err != nil {
		log.Fatalf("error: %s\n", err)
	}

	var config steamroller.Config
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
