#!/bin/bash

token=$(uuidgen)
docker compose exec mongo mongosh --eval "db.getCollection('user').insertOne({'token': '$token'})" wakizashi
echo "Added token: $token"
