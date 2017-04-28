Change log
==========

0.1.0 (2015-02-26)
------------------

Initial beta release.

* Small static go binary with no dependencies.
* Run almost any docker command (ps, run, logs, attach, kill, stop, etc...) against your cluster.
* Cluster monitoring from a single endpoint.
* TLS authentication.
* Persistent state storage on disk.
* Schedule containers based upon resources (bin packing or random).
* Schedule containers based upon constraints (`region==us-west`, `storage==ssd`).
* Easy setup: 4 discovery services available (docker hub, consul, etcd, zookeeper).

