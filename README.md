ephemera
========

A tool for lighting up a digital ocean droplet, and teaching it to burn itself
after some specified period of time.

## Options

* `name` the name of the instance to bootstrap
* `hours` the number of hours this instance should persist for

## Dependencies

ephemera assumes that the instance image you're using has `curl(1)` and `at(1)`.

## Usage

```
xenia % ephemera -name rust-dev -hours 6
2015/04/08 18:17:19 Creating droplet
2015/04/08 18:17:19 Using image debian-7-0-x64
2015/04/08 18:17:20 Droplet created
2015/04/08 18:17:20 Waiting for network information
2015/04/08 18:17:21 Droplet has been assigned a network interface
2015/04/08 18:17:21 Droplet address:  104.131.136.143
2015/04/08 18:17:21 Waiting for droplet's network interface to come up
2015/04/08 18:17:51 Droplet's network interface has come up
2015/04/08 18:17:51 Sending the shutdown script to the remote host
2015/04/08 18:17:52 Queing the instance to shutdown in 6 hours
2015/04/08 18:17:52 Successfully bootstrapped 104.131.136.143
2015/04/08 18:17:53 Instance will cost: 0.178560
```

## Security

There's a hugely yolo assumption made, and that's that the network is
trustworthy. As far as I know, there's no way to fetch the host fingerprint
from the digitalocean API.

We make a tiny attempt at coherence, wherein the hostname of the machine is
checked to match what we asked for.
