#!/bin/bash

# This script starts a docker registry and makes it available on localhost:5000

# create registry container unless it already exists
reg_name='registry'
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run -d --restart=always -p 5000:5000 --name registry registry:2
fi
