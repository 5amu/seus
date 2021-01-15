#!/bin/bash

DBGDIR="./debug"
RLSDIR="./release"

messg() { echo "[+] $1"; }
error() { echo "[-] $1"; exit 1; }
usage() { echo "Usage: deploy.sh {debug|release}"; }
deploy() {
    _scriptdir="$PWD"
    env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o "$1"
    cd "$1"
    sudo docker-compose up --build -d
    cd "$_scriptdir"
}

messg "Deploying is starting"
case "$1" in
    debug)
        messg "Executing in DEBUG mode"
        deploy "$DBGDIR" || error "Can't deploy debug"
        ;;
    release) 
        messg "Executing in RELEASE mode"
        deploy "$RLSDIR" || error "Can't deploy release"
        ;;
    *)
        usage && error "Wrong usage"
        ;;
esac
messg "deploy.sh was executed successfully"
