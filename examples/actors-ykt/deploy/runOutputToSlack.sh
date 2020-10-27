#!/bin/bash

cd $(dirname "$0")
cd ..

../../scripts/kamel-local-build.sh deploy/outputSlack.yaml
../../scripts/kamel-local-run.sh
