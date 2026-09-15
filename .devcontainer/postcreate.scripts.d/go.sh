#!/usr/bin/env bash

set -x

if [[ ${BASE_CONTAINER} == "alpine" ]]; then
    sudo apk add graphviz
fi

# Install Go packages.
echo 'set --export PATH "$HOME/go/bin" /go/bin /usr/local/go/bin $PATH' >> ~/.config/fish/config.fish
echo 'export PATH="$HOME/go/bin:/go/bin:/usr/local/go/bin:$PATH' >> ~/.bashrc
if [[ -e go.mod ]]; then
    export PATH="$HOME/go/bin:/go/bin:/usr/local/go/bin:$PATH" && go mod tidy
fi
