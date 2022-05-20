# TraefikOutboundLimiter
Traefik middleware that limits the total amount of data that passes through from the internal modules back out to the requesting clients. This plugin can effectively limit the total output from the server.

This plugin requires the use of a local API that maintains the byte totals per request source: https://github.com/AustinHellerRepo/ResetingIncrementerApi.

## Features

- Custom byte limit for each Traefik service
- Custom total byte limit for all services, effectively limiting the byte output for all services

## Usage

_Basic configuration_
```yml
testData:
  lastModified: true
  resetingIncrementerApiUrl: http://172.26.0.4:38160
  resetingIncrementerKey: the_service_name
```

## ResetingIncrementerApi

It is necessary to have a [ResetingIncrementerApi](https://github.com/AustinHellerRepo/ResetingIncrementerApi) docker container running such that it is accessible from the middleware.
