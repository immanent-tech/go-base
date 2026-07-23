#!/usr/bin/bash

set -x

if [[ ${BASE_CONTAINER} == "alpine" ]]; then
    sudo apk add python3
fi

# Install gcloud cli.
cd $HOME && \
    curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-linux-x86_64.tar.gz && \
    tar -xf google-cloud-cli-linux-x86_64.tar.gz && \
    rm google-cloud-cli-linux-x86_64.tar.gz && \
    google-cloud-sdk/install.sh --usage-reporting false --quiet --additional-components app-engine-go && \
    echo 'source /home/vscode/google-cloud-sdk/path.fish.inc' >> ~/.config/fish/config.fish

# Authenticate with gcloud.
source /home/vscode/google-cloud-sdk/path.bash.inc && \
    gcloud auth application-default login --scopes=https://www.googleapis.com/auth/androidpublisher,https://www.googleapis.com/auth/cloud-platform && \
    gcloud auth application-default set-quota-project foragd

