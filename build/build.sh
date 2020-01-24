#!/bin/bash

if [ $# -eq 0 ]; then
  echo "Need build target path"
  echo "Example: ./build.sh ../../build"
  exit 1
fi

project_root=$( cd -P "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )
echo "Project Root: ${project_root}"
TZ="Asia/Taipei" date

go build -o $1 ${project_root}/app

cp -r ${project_root}/web/static $1
cp -r ${project_root}/web/template $1
cp ../configs/appconfig.yaml $1