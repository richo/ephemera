package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

type key struct {
	bits        uint
	fingerprint string
	comment     string
	algo        string
	file        string
}

func getKey() key {
	// returns the fingerprint to use, by looking at ~/.ssh/id_rsa, but
	// superceded by EPHEMERA_KEY
	keyfile := os.Getenv("EPHEMERA_KEY")

	if keyfile == "" {
		user, err := user.Current()
		if err != nil {
			log.Fatal("Couldn't lookup current user", err)
		}
		keyfile = fmt.Sprintf("%s/.ssh/id_rsa", user.HomeDir)
	}

	out, _ := exec.Command("ssh-keygen", "-E", "md5", "-l", "-f", keyfile).Output()

	parts := strings.Split(string(out), " ")
	if len(parts) == 1 {
		log.Fatal("Couldn't parse key at ", keyfile)
	}

	fpr := strings.Split(parts[1], ":")
	if fpr[0] == "MD5" {
		fpr = fpr[1:]
	}
	fingerprint := strings.Join(fpr, ":")

	bits, err := strconv.ParseUint(parts[0], 10, 0)
	if err != nil {
		log.Fatal("Couldn't parse bits: ", err)
	}

	num_parts := len(parts)
	comment := strings.Join(parts[3:num_parts-1], " ")

	return key{
		bits:        uint(bits),
		fingerprint: fingerprint,
		comment:     comment,
		algo:        string(parts[num_parts-1]),
		file:        keyfile,
	}
}
