---
advisory: swarm-standalone
hide_from_sitemap: true
description: Swarm API response codes
keywords: docker, swarm, response, code, api
title: Swarm vs. Engine response codes
---

Docker Engine provides a REST API for making calls to the Engine daemon. Docker Swarm allows a caller to make the same calls to a cluster of Engine daemons. While the API calls are the same, the API response status codes do differ. This document explains the differences.

Four methods are included, and they are GET, POST, PUT, and DELETE.

The comparison is based on api v1.22, and all Docker Status Codes in api v1.22 are referenced from [docker-remote-api-v1.22](https://github.com/moby/moby/blob/master/docs/reference/api/docker_remote_api_v1.22.md
).

## GET

- Route:          `/_ping`
- Handler:        `ping`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
||500|

- Route:          `/events`
- Handler:        `getEvents`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|400||
||500|

- Route:          `/info`
- Handler:        `getInfo`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
||500|

- Route:          `/version`
- Handler:        `getVersion`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
||500|

- Route:          `/images/json`
- Handler:        `getImagesJSON`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|500|500|

- Route:          `/images/viz`
- Handler:        `notImplementedHandler`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|501|no this api|

- Route:          `/images/search`
- Handler:        `proxyRandom`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|500|500|

- Route:          `/images/get`
- Handler:        `getImages`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404||
|500|500|

- Route:          `/images/{name:.*}/get`
- Handler:        `proxyImageGet`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404||
|500|500|

- Route:          `/images/{name:.*}/history`
- Handler:        `proxyImage`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/images/{name:.*}/json`
- Handler:        `proxyImage`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/containers/ps`
- Handler:        `getContainersJSON`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|no this api|
|404|no this api|
|500|no this api|

- Route:          `/containers/json`
- Handler:        `getContainersJSON`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
||400|
|404||
|500|500|

- Route:          `/containers/{name:.*}/archive`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|400|400|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/export`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/changes`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/json`
- Handler:        `getContainerJSON`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/top`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/logs`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|101|101|
|200|200|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/stats`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/attach/ws`
- Handler:        `proxyHijack`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|400|400|
|404|404|
|500|500|

- Route:          `/exec/{execid:.*}/json`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/networks`
- Handler:        `getNetworks`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|400||
|500|500|

- Route:          `/networks/{networkid:.*}`
- Handler:        `getNetwork`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|

- Route:          `/volumes`
- Handler:        `getVolumes`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
||500|

- Route:          `/volumes/{volumename:.*}`
- Handler:        `getVolume`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
||500|

## POST

- Route:          `/auth`
- Handler:        `proxyRandom`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|204|204|
|500|500|

- Route:          `/commit`
- Handler:        `postCommit`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|201|201|
|404|404|
|500|500|

- Route:          `/build`
- Handler:        `postBuild`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|500|500|

- Route:          `/images/create`
- Handler:        `postImagesCreate`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|500|500|

- Route:          `/images/load`
- Handler:        `postImagesLoad`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
||200|
|201||
||500|

- Route:          `/images/{name:.*}/push`
- Handler:        `proxyImagePush`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/images/{name:.*}/tag`
- Handler:        `postTagImage`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200||
||201|
||400|
|404|404|
||409|
|500|500|

- Route:          `/containers/create`
- Handler:        `postContainersCreate`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|201|201|
|400||
||404|
||406|
|409||
|500|500|

- Route:          `/containers/{name:.*}/kill`
- Handler:        `proxyContainerAndForceRefresh`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|204|204|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/pause`
- Handler:        `proxyContainerAndForceRefresh`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|204|204|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/unpause`
- Handler:        `proxyContainerAndForceRefresh`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|204|204|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/rename`
- Handler:        `postRenameContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200||
||204|
|404|404|
|409|409|
|500|500|

- Route:          `/containers/{name:.*}/restart`
- Handler:        `proxyContainerAndForceRefresh`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|204|204|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/start`
- Handler:        `postContainersStart`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|204|204|
||304|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/stop`
- Handler:        `proxyContainerAndForceRefresh`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|204|204|
|304|304|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/update`
- Handler:        `proxyContainerAndForceRefresh`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|400|400|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/wait`
- Handler:        `proxyContainerAndForceRefresh`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|204|204|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/resize`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/attach`
- Handler:        `proxyHijack`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|101|101|
|200|200|
|400|400|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/copy`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/containers/{name:.*}/exec`
- Handler:        `postContainersExec`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|201|201|
|404|404|
||409|
|500|500|

- Route:          `/exec/{execid:.*}/start`
- Handler:        `postExecStart`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|409|409|
|500||

- Route:          `/exec/{execid:.*}/resize`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|201|201|
|404|404|
|500||

- Route:          `/networks/create`
- Handler:        `postNetworksCreate`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200||
||201|
|400||
||404|
|500|500|

- Route:          `/networks/{networkid:.*}/connect`
- Handler:        `proxyNetworkConnect`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/networks/{networkid:.*}/disconnect`
- Handler:        `proxyNetworkDisconnect`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
|500|500|

- Route:          `/volumes/create`
- Handler:        `postVolumesCreate`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200||
||201|
|400||
|500|500|

## PUT

- Route:          `/containers/{name:.*}/archive"`
- Handler:        `proxyContainer`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|400|400|
|403|403|
|404|404|
|500|500|

## DELETE

- Route:          `/containers/{name:.*}`
- Handler:        `deleteContainers`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200||
||204|
||400|
|404|404|
|500|500|

- Route:          `/images/{name:.*}`
- Handler:        `deleteImages`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|200|200|
|404|404|
||409|
|500|500|

- Route:          `/networks/{networkid:.*}`
- Handler:        `deleteNetworks`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
||200|
|204||
|404|404|
|500|500|

- Route:          `/volumes/{name:.*}"`
- Handler:        `deleteVolumes`

|Swarm Status Code|Docker Status Code|
| :-------------: | :--------------: |
|204|204|
|404|404|
||409|
|500|500|
