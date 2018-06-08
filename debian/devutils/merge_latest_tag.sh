#!/bin/bash

set -eux

source $(dirname $(readlink -f $0))/constants.sh

cd $(git rev-parse --show-toplevel)

setup_trap

git remote add upstream https://github.com/lxc/lxd-pkg-ubuntu.git || true
git fetch upstream --tags

target_tag=$(git describe --tags upstream/dpm-$UPSTREAM_VER --match 'debian/*')
target_commit=$(git show-ref --tags -s -d "$target_tag")
git checkout $OUR_BRANCH
git merge $target_commit
