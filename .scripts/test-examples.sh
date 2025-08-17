#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

filter=${1:-}
"${SCRIPT_DIR}/gen-comp.sh" ${filter} >/dev/null
echo ---

cd  "${SCRIPT_DIR}/../example"

run=0
for dir in *
do
  if [[ ! -d "${dir}" ]]
  then
    continue
  fi
  if [[ "${filter}" != "" && "${filter}" != "${dir}" ]]
  then
    continue
  fi
  run=1
  echo $dir>&2
  cd $dir
  cmd="crossplane render -r -c xr.yaml composition.yaml functions.yaml"
  if [[ -f observed.yaml ]]
  then
    cmd="${cmd} -o observed.yaml"
  fi
  ${cmd} | "${SCRIPT_DIR}/fix-output-yaml.sh" >/tmp/results.yaml
  dyff between -i --omit-header --additional-identifier type --set-exit-code src/expected.yaml  /tmp/results.yaml
  cd ..
done

if [[ "${run}" == "0" ]]
then
  echo "[ERROR] No directories matched filter $filter" >&2
  exit 2
fi
