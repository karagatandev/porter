#!/bin/bash

# Script to install the Porter CLI on macOS, Linux, and WSL

osname=""

check_prereqs() {
    command -v curl >/dev/null 2>&1 || { echo "[ERROR] curl is required to install the Porter CLI." >&2; exit 1; }
    command -v unzip >/dev/null 2>&1 || { echo "[ERROR] unzip is required to install the Porter CLI." >&2; exit 1; }
}

download_and_install() {
    check_prereqs
    
    echo "[INFO] Since the Porter CLI gets installed in /usr/local/bin, you may be asked to input your password."
    echo "[INFO] Please make sure /usr/local/bin is included in your PATH."

    porter_version="{{ .TagName }}"
    if [[ -n "$PORTER_VERSION" ]]; then
        echo "[INFO] Using $PORTER_VERSION override instead of $porter_version"
        porter_version="$PORTER_VERSION"
    fi

    curl -L "https://github.com/karagatandev/porter/releases/download/${porter_version}/porter_${porter_version}_${osname}_x86_64.zip" --output porter.zip
    unzip -o -a porter.zip
    rm porter.zip

    chmod +x ./porter
    sudo mv ./porter /usr/local/bin/porter

    command -v porter >/dev/null 2>&1 || { echo "[ERROR] There was an error installing the Porter CLI. Please try again." >&2; exit 1; }

    exit
}

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    if uname -a | grep -q '^Linux.*Microsoft'; then
        echo "[WARNING] WSL support is experimental and may result in crashes."
    fi
    osname="Linux"
    download_and_install
elif [[ "$OSTYPE" == "darwin"* ]]; then
    osname="Darwin"
    download_and_install
fi

echo "[ERROR] Unsupported operating system."
exit 1
