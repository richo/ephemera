package main

import (
	"github.com/digitalocean/godo"
	"log"
)

func createEphemeralInstance(client *godo.Client, name string) *godo.DropletRoot {

	createRequest := &godo.DropletCreateRequest{
		Name:    name,
		Region:  REGION,
		Size:    SIZE,
		SSHKeys: []godo.DropletCreateSSHKey{{Fingerprint: FINGERPRINT}},
		Image: godo.DropletCreateImage{
			Slug: IMAGE_SLUG,
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
