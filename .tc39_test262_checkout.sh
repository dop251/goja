#!/bin/sh
sha=1ba3a7c4a93fc93b3d0d7e4146f59934a896837d # this is just the commit it was last tested with
mkdir -p testdata/test262
cd testdata/test262
git init
git remote add origin https://github.com/tc39/test262.git
git fetch origin --depth=1 "${sha}"
git reset --hard "${sha}"
cd -
