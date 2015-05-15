package main

import (
	"code.google.com/p/goauth2/oauth"
	"fmt"
	"github.com/digitalocean/godo"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	key   string
	name  string
	image string
	hours int
}

// These are sane defaults for me right now, but they should be generalised or
// pulled into a config file or something
const (
	REGION = "sfo1"
	SIZE   = "4gb"
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
	log.Printf("Using image %s", cfg.image)

	key := getKey()

	log.Printf("Using key with fingerprint %s", key.fingerprint)

	droplet := createEphemeralInstance(client, cfg.name, key.fingerprint, cfg.image)
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

	log.Printf("Removing any existing host key for %s", ip_address)
	cmd := exec.Command("ssh-keygen", "-R", ip_address)
	err := cmd.Run()
	if err != nil {
		log.Fatal("Couldn't remove host key")
	}

	log.Printf("Fetching ssh host key for %s", ip_address)
	cmd = exec.Command("ssh", "-i", key.file,
		"-o", "StrictHostKeyChecking=no",
		"-o", "ControlMaster=no",
		fmt.Sprintf("root@%s", ip_address), "hostname; ssh-keyscan localhost")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("Couldn't retrieve host key")
	}

	// Assert that the machine came up ok.
	// This host key checking nonsense is ~bullshit but it's unclear how to get
	// the host key out of the digital ocean API.
	parts := strings.Split(string(output), "\n")
	if parts[0] != cfg.name {
		log.Fatal("Machine came up with a weird name, ", string(parts[0]))
	}

	host_key := parts[1]

	fh, err := os.OpenFile(os.ExpandEnv("$HOME/.ssh/known_hosts"), os.O_APPEND, 0600)
	if err != nil {
		log.Fatal("Couldn't open known hosts files")
	}
	fh.Write([]byte(host_key))
	fh.Write([]byte("\n"))

	log.Println("Sending the shutdown script to the remote host")
	cmd = exec.Command("ssh", "-i", key.file,
		"-o", "ControlMaster=no",
		fmt.Sprintf("root@%s", ip_address), "cat > .shutdown")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("Couldn't get input pipe", err)
	}
	cmd.Start()
	shutdown_cmd := shutdownCommand(cfg, droplet_id)
	pipe.Write([]byte(shutdown_cmd))
	pipe.Write([]byte("\n"))
	pipe.Close()
	cmd.Wait()

	log.Printf("Queing the instance to shutdown in %d hours", cfg.hours)
	cmd = exec.Command("ssh", "-i", key.file,
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
	log.Printf("Instance will cost: $%.2f", cost*float64(cfg.hours))
}
