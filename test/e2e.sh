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
    ${nitric} build create -s $s -p "aws"

    sudo rm -rf ${WORK_DIR}
done
