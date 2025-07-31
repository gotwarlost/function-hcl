#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd  "${SCRIPT_DIR}/../example"

for dir in *
do
  if [[ ! -d "${dir}" ]]
  then
    continue
  fi
  echo $dir
  (cd $dir && cat src/comp-template.yaml | script="$(txtar src/*hcl </dev/null | jq -sR)" envsubst | yq -P>composition.yaml)
done
