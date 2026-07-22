#!/usr/bin/bash

set -x

# Install pulumi.
curl -fsSL https://get.pulumi.com | sh \
    && echo 'set --export PULUMI_INSTALL "$HOME/.pulumi"' >> ~/.config/fish/config.fish \
    && echo 'set --export PATH $PULUMI_INSTALL/bin $PATH' >> ~/.config/fish/config.fish

