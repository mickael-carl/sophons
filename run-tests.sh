#!/usr/bin/env bash

# TODO: this should be a Go program that leverages coverage measurements.

set -uo pipefail

dir=$(mktemp -d)
ssh-keygen -f "${dir}/testing" -N "" > /dev/null

for playbook in data/playbooks/*; do

    sha=$(docker run -d -p 127.0.0.1:2222:22 sophons-testing:latest)
    docker cp "${dir}/testing.pub" "${sha}:/root/.ssh/authorized_keys" > /dev/null
    docker exec "${sha}" chown root:root /root/.ssh/authorized_keys
    ./bin/dialer -p 2222 -b bin/ -i data/inventory-testing.yaml -k "${dir}/testing" -u root "${playbook}" > /dev/null
    docker stop "${sha}" > /dev/null
    docker export -o "${dir}/sophons.tar" "${sha}"
    docker rm "${sha}" > /dev/null

    sha=$(docker run -d -p 127.0.0.1:2222:22 sophons-testing:latest)
    docker cp "${dir}/testing.pub" "${sha}:/root/.ssh/authorized_keys" > /dev/null
    docker exec "${sha}" chown root:root /root/.ssh/authorized_keys
    ansible-playbook --key-file "${dir}/testing" -u root -i data/inventory-testing.yaml --ssh-common-args="-p 2222" "${playbook}" &> /dev/null
    docker stop "${sha}" > /dev/null
    docker export -o "${dir}/ansible.tar" "${sha}"
    docker rm "${sha}" > /dev/null

    ./bin/tardiff "${dir}/ansible.tar" "${dir}/sophons.tar"

done

rm -r "${dir}"
