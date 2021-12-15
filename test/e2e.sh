#!/bin/bash
set -xe

make

nitric="${PWD}/bin/nitric"

for s in java-stack python-stack typescript-stack
do
    WORK_DIR=`mktemp -d -t nitric-cli-e2e.XXXXXXXXXX`
    echo "$s => $WORK_DIR"
    deploy_name="dep"

    # check if tmp dir was created
    if [[ ! "$WORK_DIR" || ! -d "$WORK_DIR" ]]; then
        echo "Could not create temp dir"
        exit 1
    fi
    cd ${WORK_DIR}

    ${nitric} stack create $s $s
    ${nitric} build create -s $s -p "local"
    ${nitric} deployment apply -s $s -p "local" $deploy_name

    port=$(${nitric} deployment list -s $s -o json | jq ".[0].Ports[0]")
    if [ "$(curl localhost:$port/examples)" != "[]" ]
    then
        echo curl localhost:$port/examples returned unexpected result
        curl localhost:$port/examples
        exit 1
    fi

    ${nitric} deployment delete -s $s -p "local" $deploy_name

    sudo rm -rf ${WORK_DIR}
done
