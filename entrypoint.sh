#!/usr/bin/env bash

set -e

exec /loss-prevention-service -r --profile=docker --confdir=/res
