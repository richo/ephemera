package main

import (
	"code.google.com/p/goauth2/oauth"
	"flag"
	"fmt"
	"github.com/digitalocean/godo"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"os/user"
	"strings"
	"time"
)

type Config struct {
	key   string
	name  string
	hours int
}

// These are sane defaults for me right now, but they should be generalised or
// pulled into a config file or something
const (
	REGION      = "sfo1"
	SIZE        = "2gb"
	IMAGE_SLUG  = "debian-7-0-x64"
	FINGERPRINT = "91:ff:af:1c:e2:0c:5e:b7:dd:8d:6c:27:0d:e6:20:63"
)

func parse_flags() *Config {
	name := flag.String("name", "", "instance name")
	hours := flag.Int("hours", 24, "hours the instance should persist")
	flag.Parse()

	if *name == "" {
		return nil
	}

	return &Config{
		"",
		*name,
		*hours,
	}
}

func get_token() string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	dat, err := ioutil.ReadFile(dir + "/.ephemera")
	if err != nil {
		log.Fatal("Couldn't read token file")
	}
	return string(dat)
}

func get_config() *Config {
	base := parse_flags()
	if base == nil {
		return nil
	}

	base.key = get_token()

	return base
}

func shutdown_command(cfg *Config, id int) string {
	return fmt.Sprintf("curl -X DELETE -H 'Content-Type: application/json' -H 'Authorization: Bearer %s' 'https://api.digitalocean.com/v2/droplets/%d'",
		cfg.key, id)
}

func main() {
	cfg := get_config()
	if cfg == nil {
		log.Fatal("Couldn't parse flags")
	}

	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: cfg.key},
	}
	client := godo.NewClient(t.Client())
	// Specialcase to dump all instance types. This is silly
	if cfg.name == "?" {
		list_all_images(client)
		return
	}

	droplet := create_ephemeral_instance(client, cfg.name)
	droplet_id := droplet.Droplet.ID

	log.Println("Waiting for droplet to come up")
	var ip_address string
	for {
		droplet, _, err := client.Droplets.Get(droplet_id)
		if err != nil {
			log.Fatal("Couldn't get droplet info", err)
		}
		if len(droplet.Droplet.Networks.V4) > 0 {
			// Droplet has probably come up
			ip_address = droplet.Droplet.Networks.V4[0].IPAddress
			log.Println("Droplet appears to be up, boostrapping")
			log.Println("Droplet address: ", ip_address)
			break
		}

		log.Printf("Sleeping for 5s")
		time.Sleep(5 * time.Second)
	}

	log.Println("Waiting for droplet's sshd to start")
	addr := fmt.Sprintf("%s:22", ip_address)
	for {
		// The trick here is that even with our timeout, there's a window in
		// which the connection will be refused.
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			time.Sleep(5 * time.Second)
			log.Println("Retrying tcp probe")
		} else {
			conn.Close()
			break
		}
	}

	// Assert that the machine came up ok.
	// This host key checking nonsense is ~bullshit but it's unclear how to get
	// the host key out of the digital ocean API.

	out, err := exec.Command("ssh", "-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("root@%s", ip_address), "hostname").Output()

	if err != nil {
		log.Fatal("Couldn't fetch hostname", err)
	}

	if strings.Trim(string(out), "\n\r ") != cfg.name {
		log.Fatal("Machine came up with a weird name, ", string(out))
	}

	// Copy the script to shut the machine down up
	log.Println("Sending the shutdown script to the remote host")
	cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no",
		"-o", "ControlMaster=no",
		fmt.Sprintf("root@%s", ip_address), "cat > .shutdown")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("Couldn't get input pipe", err)
	}
	cmd.Start()
	shutdown_cmd := shutdown_command(cfg, droplet_id)
	log.Println("Sending the command")
	pipe.Write([]byte(shutdown_cmd))
	pipe.Write([]byte("\n"))
	pipe.Close()
	log.Println("Waiting on the remote process to end")
	cmd.Wait()

	log.Println("Queing the instance to shutdown in 24 hours")
	cmd = exec.Command("ssh", "-o", "StrictHostKeyChecking=no",
		"-o", "ControlMaster=no",
		fmt.Sprintf("root@%s", ip_address),
		fmt.Sprintf("at -f .shutdown now + %d hours", cfg.hours))
	cmd.Wait()

	log.Printf("Successfully bootstrapped", ip_address)
}

func create_ephemeral_instance(client *godo.Client, name string) *godo.DropletRoot {

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
		log.Fatal("Something bad happened: %s", err)
	}

	return newDroplet
}

func list_all_images(client *godo.Client) {
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
