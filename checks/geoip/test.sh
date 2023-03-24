#! /bin/bash

. ../../.env

export GOOGLE_MAPS_API_KEY
export MAXMIND_USER_ID
export MAXMIND_LICENSE_KEY

go test

