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
      - traefik.http.middlewares.my_middleware.plugin.traefikoutboundlimiter.resetingIncrementerApiUrl=http://resetingincrementerapi-web-1:38160
      - traefik.http.middlewares.my_middleware.plugin.traefikoutboundlimiter.resetingIncrementerKey=the_service_name
networks:
  default:
    name: traefik_router_network
    external: true
```
Important:
- The url "http://resetingincrementerapi-web-1" is the default container name when starting the docker-compose.yml from the ResetingIncrementerApi project.
- The key "the_service_name" will need to be one of the keys specified in the running ResetingIncrementerApi settings.ini file.

_Starting docker container_
```sh
docker network inspect traefik_router_network >/dev/null 2>&1 || \
  docker network create --driver bridge traefik_router_network
docker-compose up -d
```

## ResetingIncrementerApi

It is necessary to have a [ResetingIncrementerApi](https://github.com/AustinHellerRepo/ResetingIncrementerApi) docker container running such that it is accessible from the middleware.

- Setup such that it is accessible from Traefik services
  - This generally can be solved by having the ResetingIncrementerApi in the same docker network as your services
- Determine reset interval to establish monthly byte limit or "after X seconds" byte limit
```yml
[Timing]
Interval = day_of_month
Value = 1
```

If you're having issues connecting to the ResetingIncrementerApi docker container, make sure that it's running in the same network. Try a "docker network inspect traefik_router_network" call in order to see that everything is available to each other and what the IP address is for the API.

## Adding New Service
Steps:
- Update the ResetingIncrementerApi setting.ini file to contain a new key limit for the new service
```yml
[KeyLimits]
new_service = 123
; the new_service can output 123 bytes before being restricted
```
- Set the new service's resetingIncrementerKey label value to that same key
```yml
labels:
  - traefik.http.middlewares.my_middleware.plugin.traefikoutboundlimiter.resetingIncrementerApiUrl=http://resetingincrementerapi-web-1:38160
  - traefik.http.middlewares.my_middleware.plugin.traefikoutboundlimiter.resetingIncrementerKey=new_service
```
- Restart the ResetingIncrementerApi docker container
  - This is why it is essential to have a mounted volume setup for the "data" directory
  - Restarting this container is the only source of downtime for existing services
- Start the new service at your convenience
