#!/usr/bin/bash

set -x

# Install oh-my-posh.
mkdir -p ~/.local/bin && curl -s https://ohmyposh.dev/install.sh | bash -s
# Set up shells to use oh-my-posh.
mkdir -p ~/.config/fish \
    && echo "~/.local/bin/oh-my-posh init fish | source" >>~/.config/fish/config.fish \
    && echo 'eval "$(~/.local/bin/oh-my-posh init bash)""' >>~/.bashrc

exit 0
