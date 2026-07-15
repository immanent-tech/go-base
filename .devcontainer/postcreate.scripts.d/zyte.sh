#!/usr/bin/bash

set -x

# Add the Zyte CA cert.
sudo curl -L -O /usr/local/share/ca-certificates/zyte-ca.crt https://docs.zyte.com/_static/zyte-ca.crt && \
    sudo update-ca-certificates

exit 0
