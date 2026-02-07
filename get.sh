#!/usr/bin/env sh

# Adapted from https://github.com/alexellis/k3sup/blob/master/get.sh.
# See https://github.com/alexellis/k3sup/LICENSE
#########################
# Repo specific content #
#########################

# cSpell: words armv armhf

export VERIFY_CHECKSUM=1
export ALIAS_NAME=""
export OWNER=karmafun
export REPO=karmafun
export BIN_LOCATION="/usr/local/bin"
export SUCCESS_CMD="$BIN_LOCATION/$REPO --version"

###############################
# Content common across repos #
###############################

version=$(curl -sI https://github.com/$OWNER/$REPO/releases/latest | grep -i "location:" | awk -F"/" '{ printf "%s", $NF }' | tr -d '\r')
if [ ! "$version" ]; then
    echo "Failed while attempting to install $REPO. Please manually install:"
    echo ""
    echo "1. Open your web browser and go to https://github.com/$OWNER/$REPO/releases"
    echo "2. Download the latest release for your platform. Call it '$REPO'."
    echo "3. chmod +x ./$REPO"
    echo "4. mv ./$REPO $BIN_LOCATION"
    if [ -n "$ALIAS_NAME" ]; then
        echo "5. ln -sf $BIN_LOCATION/$REPO /usr/local/bin/$ALIAS_NAME"
    fi
    exit 1
fi

hasCli() {

    if ! command -v curl >/dev/null 2>&1; then
        echo "You need curl to use this script."
        exit 1
    fi
}

checkHash() {

    sha_cmd="sha256sum"

    if ! command -v sha256sum >/dev/null 2>&1; then
        if command -v shasum >/dev/null 2>&1; then
            sha_cmd="shasum -a 256"
        else
            echo "Neither sha256sum nor shasum is available. Cannot verify binary checksum."
            return
        fi
    fi

    targetFileDir=${targetFile%/*}
    sha_url="https://github.com/$OWNER/$REPO/releases/download/$version/SHA256SUMS"

    if ! (cd "$targetFileDir" && curl -sSL "$sha_url" | grep "$fileName" | $sha_cmd -c >/dev/null); then
        rm "$targetFile"
        echo "Binary checksum didn't match. Exiting"
    fi
}

getPackage() {
    uname=$(uname)
    userid=$(id -u)

    suffix=""
    case $uname in
    "Darwin")
        arch=$(uname -m)
        case $arch in
        "x86_64")
            suffix="darwin_amd64"
            ;;
        "arm64")
            suffix="darwin_arm64"
            ;;
        esac
        ;;

    "MINGW"*)
        arch=$(uname -m)
        case $arch in
        "aarch64")
            suffix="windows_arm64"
            ;;
        "i386")
            suffix="windows_386"
            ;;
        "x86_64")
            suffix="windows_amd64"
            ;;
        esac

        BIN_LOCATION="$HOME/bin"
        mkdir -p "$BIN_LOCATION"

        ;;
    "Linux")
        arch=$(uname -m)
        case $arch in
        "aarch64")
            suffix="linux_arm64"
            ;;
        "armv6l" | "armv7l")
            suffix="linux_armhf"
            ;;
        "i386")
            suffix="linux_386"
            ;;
        "x86_64")
            suffix="linux_amd64"
            ;;
        esac
        ;;
    esac

    fileName="${REPO}_${version}_$suffix"
    targetFile="/tmp/$fileName"

    if [ "$userid" != "0" ]; then
        targetFile="$(pwd)/$fileName"
    fi

    if [ -e "$targetFile" ]; then
        rm "$targetFile"
    fi

    url="https://github.com/$OWNER/$REPO/releases/download/$version/$fileName"
    echo "Downloading package $url as $targetFile"

    if curl -sSL "$url" --output "$targetFile"; then

        if [ "$VERIFY_CHECKSUM" = "1" ]; then
            checkHash
        fi

        chmod +x "$targetFile"

        echo "Download complete."

        if [ ! -w "$BIN_LOCATION" ]; then

            echo
            echo "============================================================"
            echo "  The script was run as a user who is unable to write"
            echo "  to $BIN_LOCATION. To complete the installation the"
            echo "  following commands may need to be run manually."
            echo "============================================================"
            echo
            echo "  sudo cp $REPO$suffix $BIN_LOCATION/$REPO"

            if [ -n "$ALIAS_NAME" ]; then
                echo "  sudo ln -sf $BIN_LOCATION/$REPO $BIN_LOCATION/$ALIAS_NAME"
            fi

            echo

        else

            echo
            echo "Running with sufficient permissions to attempt to move $REPO to $BIN_LOCATION"

            if [ ! -w "$BIN_LOCATION/$REPO" ] && [ -f "$BIN_LOCATION/$REPO" ]; then

                echo
                echo "================================================================"
                echo "  $BIN_LOCATION/$REPO already exists and is not writeable"
                echo "  by the current user.  Please adjust the binary ownership"
                echo "  or run sh/bash with sudo."
                echo "================================================================"
                echo
                exit 1

            fi

            if mv "$targetFile" "$BIN_LOCATION/$REPO"; then
                echo "New version of $REPO installed to $BIN_LOCATION"
            fi

            if [ -e "$targetFile" ]; then
                rm "$targetFile"
            fi

            if [ -n "$ALIAS_NAME" ]; then
                if [ "$(which "$ALIAS_NAME")" ]; then
                    echo "There is already a command '$ALIAS_NAME' in the path, NOT creating alias"
                else
                    if [ ! -L "$BIN_LOCATION/$ALIAS_NAME" ]; then
                        ln -s "$BIN_LOCATION/$REPO" "$BIN_LOCATION/$ALIAS_NAME"
                        echo "Creating alias '$ALIAS_NAME' for '$REPO'."
                    fi
                fi
            fi

            ${SUCCESS_CMD}
        fi
    fi
}

hasCli
getPackage
