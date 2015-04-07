package main

import (
	"code.google.com/p/goauth2/oauth"
	"fmt"
	"github.com/digitalocean/godo"
	"log"
	"net"
	"os/exec"
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

func shutdownCommand(cfg *Config, id int) string {
	return fmt.Sprintf("curl -X DELETE -H 'Content-Type: application/json' -H 'Authorization: Bearer %s' 'https://api.digitalocean.com/v2/droplets/%d'",
		cfg.key, id)
}

func main() {
	cfg := getConfig()
	if cfg == nil {
		log.Fatal("Couldn't parse flags")
	}

	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: cfg.key},
	}
	client := godo.NewClient(t.Client())
	// Specialcase to dump all instance types. This is silly
	if cfg.name == "?" {
		listAllImages(client)
		return
	}

	log.Println("Creating droplet")

	droplet := createEphemeralInstance(client, cfg.name)
	droplet_id := droplet.Droplet.ID

	log.Println("Droplet created")
	log.Println("Waiting for network information")
	var ip_address string
	for {
		droplet, _, err := client.Droplets.Get(droplet_id)
		if err != nil {
			log.Fatal("Couldn't get droplet info", err)
		}
		if len(droplet.Droplet.Networks.V4) > 0 {
			// Droplet has probably come up
			ip_address = droplet.Droplet.Networks.V4[0].IPAddress
			log.Println("Droplet has been assigned a network interface")
			log.Println("Droplet address: ", ip_address)
			break
		}

		log.Printf("Sleeping for 5s")
		time.Sleep(5 * time.Second)
	}

	log.Println("Waiting for droplet's network interface to come up")
	addr := fmt.Sprintf("%s:22", ip_address)
	for {
		// The trick here is that even with our timeout, there's a window in
		// which the connection will be refused.
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			time.Sleep(5 * time.Second)
		} else {
			conn.Close()
			break
		}
	}
	log.Println("Droplet's network interface has come up")

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
	shutdown_cmd := shutdownCommand(cfg, droplet_id)
	log.Println("Sending the command")
	pipe.Write([]byte(shutdown_cmd))
	pipe.Write([]byte("\n"))
	pipe.Close()
	log.Println("Waiting on the remote process to end")
	cmd.Wait()

	log.Printf("Queing the instance to shutdown in %d hours", cfg.hours)
	cmd = exec.Command("ssh", "-o", "StrictHostKeyChecking=no",
		"-o", "ControlMaster=no",
		fmt.Sprintf("root@%s", ip_address),
		fmt.Sprintf("at -f .shutdown now + %d hours", cfg.hours))
	_, err = cmd.Output()
	if err != nil {
		log.Fatal("Couldn't configure shutdown")
	}

	log.Println("Successfully bootstrapped", ip_address)

	cost, err := costPerHour(SIZE, client)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Instance will cost: %f", cost*float64(cfg.hours))
}
