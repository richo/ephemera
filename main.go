package main

import (
	"code.google.com/p/goauth2/oauth"
	"flag"
	"github.com/digitalocean/godo"
	"log"
)

type Config struct {
	key  string
	name string
}

// These are sane defaults for me right now, but they should be generalised or
// pulled into a config file or something
const (
	REGION     = "sfo1"
	SIZE       = "2g"
	IMAGE_SLUG = "3102387" // Debian 7.0 x64
)

func parse_flags() *Config {
	key := flag.String("key", "", "api key")
	name := flag.String("name", "", "instance name")
	flag.Parse()

	if *name == "" || *key == "" {
		return nil
	}

	return &Config{
		*key,
		*name,
	}
}

func main() {
	cfg := parse_flags()
	if cfg == nil {
		log.Fatal("Couldn't parse flags")
	}

	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: cfg.key},
	}

	client := godo.NewClient(t.Client())

	createRequest := &godo.DropletCreateRequest{
		Name:   cfg.name,
		Region: REGION,
		Size:   SIZE,
		Image: godo.DropletCreateImage{
			Slug: IMAGE_SLUG,
		},
	}

	newDroplet, _, err := client.Droplets.Create(createRequest)

	if err != nil {
		log.Fatal("Something bad happened: %s", err)
	}

	log.Printf("%s", newDroplet)
}
