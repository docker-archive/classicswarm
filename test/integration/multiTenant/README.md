# Swarm Multi-Tenant Integration Tests

Integration tests provide end-to-end testing of Swarm API's supported
in multi-tenant environment.

Integration tests are written in *bash* using the
[bats](https://github.com/sstephenson/bats) framework.

## Setup

1. Setup Swarm development environment plus [bats](https://github.com/sstephenson/bats#installing-bats-from-source)

2. Set tenant header in: $HOME/.docker/config.json
```
{
       "HttpHeaders": {
                       "X-Auth-Token": "<user_name>",
                       "X-Auth-TenantId": "<user_name>"
       }
}
```

## Running integration tests

Run on your host:

```
$ bats test/integration/multiTenant/
```

In order to do that, you will need to setup a full development environment plus
[bats](https://github.com/sstephenson/bats#installing-bats-from-source)


