#!/bin/bash

cmd="$@";

docker run \
	--network="host" \
	-v $HOME/.pulumi:/root/.pulumi \
	-v $HOME:$HOME \
	-v $HOME/.aws:/root/.aws \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v $(pwd):/app \
	-e PULUMI_HOME=/root/.pulumi \
	-e PULUMI_CONFIG_PASSPHRASE_FILE=$PULUMI_CONFIG_PASSPHRASE_FILE \
	-e PULUMI_DEBUG_COMMANDS=true \
	-w /app \
    pulumi/pulumi:3.43.1 \
    $cmd