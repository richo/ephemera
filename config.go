package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os/user"
	"strings"
)

func parse_flags() *Config {
	name := flag.String("name", "", "instance name")
	hours := flag.Int("hours", 24, "hours the instance should persist")
	image := flag.String("image", "debian-7-0-x64", "image slug")
	size := flag.String("size", "2gb", "size of instance")
	bootstrap := flag.String("bootstrap", "", "shell script to bootstrap with")
	flag.Parse()

	if *name == "" {
		return nil
	}

	return &Config{
		"",
		*name,
		*image,
		*hours,
		*size,
		*bootstrap,
	}
}

func getToken() string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	dat, err := ioutil.ReadFile(dir + "/.ephemera")
	if err != nil {
		log.Fatal("Couldn't read token file")
	}
	return strings.Trim(string(dat), "\r\n ")
}

func getConfig() *Config {
	base := parse_flags()
	if base == nil {
		return nil
	}

	base.key = getToken()

	return base
}
