#!/usr/bin/bash

set -x

if [[ ${BASE_CONTAINER} == "alpine" ]]; then
    sudo apk add npm
fi

# Update JS packages with bun.
if [[ -e ./package.json ]]; then
    npm clean-install || exit -1
fi
echo 'set --export PATH "/workspace/node_modules/.bin" $PATH' >> ~/.config/fish/config.fish
echo 'export PATH=/workspace/node_modules/.bin:${PATH}' >> ~/.bashrc
