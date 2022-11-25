#!/bin/bash

# This script runs a pulumi as a container compatible with
# the pulumi automation API.

# This script should only be run in the absence of a local pulumi installation

cmd="$@";

docker run \
    --rm \
	--network="host" \
	-v $HOME/.pulumi:/root/.pulumi \
	-v $HOME:$HOME \
	-v $HOME/.aws:/root/.aws \
    -v $HOME/.config/gclod:/root/.config/gcloud \
    -v $HOME/.azure:/root/.azure \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v $(pwd):/app \
	-e PULUMI_HOME=/root/.pulumi \
	-e PULUMI_CONFIG_PASSPHRASE_FILE=$PULUMI_CONFIG_PASSPHRASE_FILE \
	-e PULUMI_DEBUG_COMMANDS=true \
	-w /app \
    pulumi/pulumi \
    $cmd