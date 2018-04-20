#!/bin/bash

set -eux

source $(dirname $(readlink -f $0))/constants.sh

pushd $REPO_ROOT

setup_trap

git pull upstream --tags

target_tag=$(git describe --tags upstream/dpm-$UPSTREAM_VER --match 'debian/*')
target_commit=$(git show-ref --tags -s -d "$target_tag")
git checkout $OUR_BRANCH
git merge $target_commit

popd
