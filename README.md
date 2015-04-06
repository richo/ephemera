ephemera
========

A tool for lighting up a digital ocean droplet, and teaching it to burn itself
after some specified period of time.

## Options

* `name` the name of the instance to bootstrap
* `hours` the number of hours this instance should persist for

## Dependencies

ephemera assumes that the instance image you're using has `curl(1)` and `at(1)`.
