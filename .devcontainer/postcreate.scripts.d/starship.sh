#!/usr/bin/bash

set -x

# Install starship
cd /tmp && curl -sS https://starship.rs/install.sh | sh -s -- -y || exit -1

