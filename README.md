# TraefikOutboundLimiter
Traefik middleware that limits the total amount of data that passes through from the internal modules back out to the requesting clients. This plugin can effectively limit the total output from the server.

This plugin requires the use of a local API that maintains the byte totals per request source: https://github.com/AustinHellerRepo/ResetingIncrementerApi.
