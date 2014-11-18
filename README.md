# swarm: docker clustering

## Example usage

```bash
# create a cluster
$ swarm create
6856663cdefdec325839a4b7e1de38e8

# on each of your nodes, start the swarm agent
$ docker run -d -p 4243:4243 swarm join --token=6856663cdefdec325839a4b7e1de38e8 --addr=<docker_daemon_ip1:4243>
$ docker run -d -p 4243:4243 swarm join --token=6856663cdefdec325839a4b7e1de38e8 --addr=<docker_daemon_ip2:4243>
$ docker run -d -p 4243:4243 swarm join --token=6856663cdefdec325839a4b7e1de38e8 --addr=<docker_daemon_ip3:4243>
...

# start the manager on any machine or your laptop
$ docker run -d -p 4243:4243 swarm manage --token=6856663cdefdec325839a4b7e1de38e8

# use the regular docker cli
$ docker -H <ip:4243> ps 
$ docker -H <ip:4243> run ... 
$ docker -H <ip:4243> info
...

# list nodes in your cluster
$ swarm list --token=6856663cdefdec325839a4b7e1de38e8
http://<docker_daemon_ip1:4243>
http://<docker_daemon_ip2:4243>
http://<docker_daemon_ip3:4243>
```

## Creators

**Andrea Luzzardi**

- <http://twitter.com/aluzzardi>
- <http://github.com/aluzzardi>

**Victor Vieux**

- <http://twitter.com/vieux>
- <http://github.com/vieux>

## Copyright and license

Code and documentation copyright 2014 Docker, inc. Code released under the Apache 2.0 license.
Docs released under Creative commons.

