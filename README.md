# TraefikOutboundLimiter
Traefik middleware that limits the total amount of data that passes through from the internal modules back out to the requesting clients. This plugin can effectively limit the total output from the server.

This plugin requires the use of a local API that maintains the byte totals per request source: https://github.com/AustinHellerRepo/ResetingIncrementerApi.

## Features

- Custom byte limit for each Traefik service
- Custom total byte limit for all services, effectively limiting the byte output for all services
- Resets the recorded outbound byte totals after a specific interval (number of seconds or after the Xth day of the month)
- Returns 409 HTTP error code when limit is reached

## Usage

_Basic configuration_
```yml
testData:
  lastModified: true
  resetingIncrementerApiUrl: http://172.26.0.4:38160
  resetingIncrementerKey: the_service_name
```

_docker-compose example_
```yml
version: '3.8'

services:
  web:
    build:
      context: .
      dockerfile: docker/Dockerfile
    command: gunicorn api.main:app --bind 0.0.0.0:38155 -w 4 -k uvicorn.workers.UvicornWorker
    expose:
      - 38155
    labels:
      - traefik.enable=true
      - traefik.http.routers.my_api.rule=Host(`subdomain.domain.com`)
      - traefik.http.routers.my_api.entrypoints=web
      - traefik.http.routers.my_api.middlewares=my_middleware
      - traefik.http.middlewares.my_middleware.plugin.traefikoutboundlimiter.lastModified=true
      - traefik.http.middlewares.my_middleware.plugin.traefikoutboundlimiter.resetingIncrementerApiUrl=http://172.26.0.4:38160
      - traefik.http.middlewares.my_middleware.plugin.traefikoutboundlimiter.resetingIncrementerKey=the_service_name
networks:
  default:
    name: traefik_router_network
    external: true
```

## ResetingIncrementerApi

It is necessary to have a [ResetingIncrementerApi](https://github.com/AustinHellerRepo/ResetingIncrementerApi) docker container running such that it is accessible from the middleware.

- Setup such that it is accessible from Traefik services
- Determine reset interval to establish monthly byte limit or "after X seconds" byte limit
