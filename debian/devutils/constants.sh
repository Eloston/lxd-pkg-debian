#!/bin/bash

# Source this script get the variables

UPSTREAM_VER=bionic
OUR_BRANCH=stretch-backports

REPO_ROOT=$(readlink -f $(dirname $(readlink -f $0))/../../)

function setup_trap {
    function abort_pushd {
        popd
    }
    trap abort_pushd SIGINT
}
