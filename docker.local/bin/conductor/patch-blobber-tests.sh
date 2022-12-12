#!/bin/sh

sed -i 's/POSTGRES_USER: postgres/&\n      POSTGRES_PASSWORD: secret/' docker.local/b0docker-compose.yml
