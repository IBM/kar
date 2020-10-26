#!/bin/bash

# Script to build a Camel Slack integration locally

SCRIPTDIR=../../scripts
WORKSPACE=slack-integration

# Clear any previous attempts

rm -rf $WORKSPACE

# Create workspace directory where all intermediate integration files will be stored

mkdir -p $WORKSPACE

# Create a directory to store all the integration properties and populate it with a viable Slack webhook URL

mkdir -p $WORKSPACE/properties

echo camel.component.slack.webhookUrl=$KAR_SLACK_WEBHOOK > "$WORKSPACE"/properties/slack.properties
if [ -z $KAR_SLACK_WEBHOOK ]; then
  echo "Warning: please set property camel.component.slack.webhookUrl in $WORKSPACE/properties/slack.properties"
fi

# Build the integration locally

./$SCRIPTDIR/kamel-local-build.sh --workspace $WORKSPACE output.yaml

# Run the integration locally

./$SCRIPTDIR/kamel-local-run.sh --workspace $WORKSPACE
