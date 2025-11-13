#!/usr/bin/env bash

# TODO: this should be a Go program that leverages coverage measurements.

set -euo pipefail

dir=$(mktemp -d)
ssh-keygen -f "${dir}/testing" -N "" > /dev/null

HOST="127.0.0.1"
PORT="2222"
KNOWN_HOSTS="${HOME}/.ssh/known_hosts"
SOPHONS_TESTING_IMAGE=${SOPHONS_TESTING_IMAGE:-"sophons-testing:latest"}

mkdir -p "${HOME}/.ssh"
touch "${KNOWN_HOSTS}"

setup_known_hosts() {
    if ! grep -q "\[${HOST}\]:${PORT}" "${KNOWN_HOSTS}" 2>/dev/null; then
        ssh-keyscan -p "${PORT}" "${HOST}" 2> /dev/null | sed "s/^${HOST}/[${HOST}]:${PORT}/" >> "${KNOWN_HOSTS}"
    fi
}

for playbook in data/playbooks/playbook*.yaml; do
    echo "running ${playbook}"

    sha=$(docker run -d -p "127.0.0.1:${PORT}:22" "${SOPHONS_TESTING_IMAGE}")
    setup_known_hosts
    docker cp "${dir}/testing.pub" "${sha}:/root/.ssh/authorized_keys" > /dev/null
    docker exec "${sha}" chown root:root /root/.ssh/authorized_keys
    ./bin/dialer -p "${PORT}" -b bin/ -i data/inventory-testing.yaml -k "${dir}/testing" -u root "${playbook}" > /dev/null
    docker stop "${sha}" > /dev/null
    docker export -o "${dir}/sophons.tar" "${sha}"
    docker rm "${sha}" > /dev/null

    sha=$(docker run -d -p "127.0.0.1:${PORT}:22" "${SOPHONS_TESTING_IMAGE}")
    setup_known_hosts
    docker cp "${dir}/testing.pub" "${sha}:/root/.ssh/authorized_keys" > /dev/null
    docker exec "${sha}" chown root:root /root/.ssh/authorized_keys
    ansible-playbook --key-file "${dir}/testing" -u root -i data/inventory-testing.yaml --ssh-common-args="-p 2222" "${playbook}" &> /dev/null
    docker stop "${sha}" > /dev/null
    docker export -o "${dir}/ansible.tar" "${sha}"
    docker rm "${sha}" > /dev/null

    ./bin/tardiff "${dir}/ansible.tar" "${dir}/sophons.tar"

done

rm -r "${dir}"
