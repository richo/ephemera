ephemera
========

A tool for lighting up a digital ocean droplet, and teaching it to burn itself
after some specified period of time.

## Options

* `name` the name of the instance to bootstrap
* `hours` the number of hours this instance should persist for

## Dependencies

ephemera assumes that the instance image you're using has `curl(1)` and `at(1)`.

## Security

There's a hugely yolo assumption made, and that's that the network is
trustworthy. As far as I know, there's no way to fetch the host fingerprint
from the digitalocean API.

We make a tiny attempt at coherence, wherein the hostname of the machine is
checked to match what we asked for.
