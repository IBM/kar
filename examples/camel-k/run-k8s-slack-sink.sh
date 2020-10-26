#!/bin/bash

../../scripts/kamel-k8s-run.sh --name=sink output.yaml -p camel.component.slack.webhookUrl=${SLACK_KAR_OUTPUT_WEBHOOK} --dev
