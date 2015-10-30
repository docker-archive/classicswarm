---
layout: "docs"
page_title: "Service Definition"
sidebar_current: "docs-agent-services"
description: |-
  One of the main goals of service discovery is to provide a catalog of available services. To that end, the agent provides a simple service definition format to declare the availability of a service and to potentially associate it with a health check. A health check is considered to be application level if it associated with a service. A service is defined in a configuration file or added at runtime over the HTTP interface.
---

# Services

One of the main goals of service discovery is to provide a catalog of available
services. To that end, the agent provides a simple service definition format
to declare the availability of a service and to potentially associate it with
a health check. A health check is considered to be application level if it
associated with a service. A service is defined in a configuration file
or added at runtime over the HTTP interface.

## Service Definition

A service definition that is a script looks like:

```javascript
{
  "service": {
    "name": "redis",
    "tags": ["master"],
    "address": "127.0.0.1",
    "port": 8000,
    "checks": [
      {
        "script": "/usr/local/bin/check_redis.py",
        "interval": "10s"
      }
    ]
  }
}
```

A service definition must include a `name` and may optionally provide
an `id`, `tags`, `address`, `port`, and `check`.  The `id` is set to the `name` if not
provided. It is required that all services have a unique ID per node, so if names
might conflict then unique IDs should be provided.

The `tags` property is a list of values that are opaque to Consul but can be used to
distinguish between "master" or "slave" nodes, different versions, or any other service
level labels.

The `address` field can be used to specify a service-specific IP address. By
default, the IP address of the agent is used, and this does not need to be provided.
The `port` field can be used as well to make a service-oriented architecture
simpler to configure; this way, the address and port of a service can
be discovered.

A service can have an associated health check. This is a powerful feature as
it allows a web balancer to gracefully remove failing nodes, a database
to replace a failed slave, etc. The health check is strongly integrated in
the DNS interface as well. If a service is failing its health check or a
node has any failing system-level check, the DNS interface will omit that
node from any service query.

The check must be of the script, HTTP, or TTL type. If it is a script type, `script`
and `interval` must be provided. If it is a HTTP type, `http` and
`interval` must be provided. If it is a TTL type, then only `ttl` must be
provided. The check name is automatically generated as
`service:<service-id>`. If there are multiple service checks registered, the
ID will be generated as `service:<service-id>:<num>` where `<num>` is an
incrementing number starting from `1`.

Note: there is more information about [checks here](/docs/agent/checks.html). 

To configure a service, either provide it as a `-config-file` option to the
agent or place it inside the `-config-dir` of the agent. The file must
end in the ".json" extension to be loaded by Consul. Check definitions can
also be updated by sending a `SIGHUP` to the agent. Alternatively, the
service can be registered dynamically using the [HTTP API](/docs/agent/http.html).

## Multiple Service Definitions

Multiple services definitions can be provided at once using the `services`
(plural) key in your configuration file.

```javascript
{
  "services": [
    {
      "id": "red0",
      "name": "redis",
      "tags": [
        "master"
      ],
      "address": "127.0.0.1",
      "port": 6000,
      "checks": [
        {
          "script": "/bin/check_redis -p 6000",
          "interval": "5s",
          "ttl": "20s"
        }
      ]
    },
    {
      "id": "red1",
      "name": "redis",
      "tags": [
        "delayed",
        "slave"
      ],
      "address": "127.0.0.1",
      "port": 7000,
      "checks": [
        {
          "script": "/bin/check_redis -p 7000",
          "interval": "30s",
          "ttl": "60s"
        }
      ]
    },
    ...
  ]
}
```
