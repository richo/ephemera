package main

import (
	"errors"
	"fmt"
	"github.com/digitalocean/godo"
	"log"
)

func costPerHour(slug string, client *godo.Client) (float64, error) {
	// cargo cult DO's insane loop construct
	opt := &godo.ListOptions{}
	for {
		images, resp, err := client.Sizes.List(opt)
		if err != nil {
			log.Fatal(err)
		}

		// append the current page's droplets to our list
		for _, d := range images {
			if d.Slug == slug {
				return d.PriceHourly, nil
			}
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			log.Fatal(err)
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return 0, errors.New(fmt.Sprintf("Couldn't find slug %s", slug))
}

func createEphemeralInstance(client *godo.Client, name, fingerprint, image_slug string) *godo.DropletRoot {

	createRequest := &godo.DropletCreateRequest{
		Name:    name,
		Region:  REGION,
		Size:    SIZE,
		SSHKeys: []godo.DropletCreateSSHKey{{Fingerprint: fingerprint}},
		Image: godo.DropletCreateImage{
			Slug: image_slug,
		},
	}

	newDroplet, _, err := client.Droplets.Create(createRequest)

	if err != nil {
		log.Fatal("Something bad happened:", err)
	}

	return newDroplet
}

func listAllImages(client *godo.Client) {
	opt := &godo.ListOptions{}
	for {
		images, resp, err := client.Images.List(opt)
		if err != nil {
			log.Fatal(err)
		}

		// append the current page's droplets to our list
		for _, d := range images {
			log.Printf("- %s\n", d)
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			log.Fatal(err)
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

}
