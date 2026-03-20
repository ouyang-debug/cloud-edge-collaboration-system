#!/bin/bash

# Define configuration items
LINK_NAME="run"
# Prefix directory for target files, default is current directory (.), can be modified as needed
TARGET_DIR="./agents"

# Script usage help information
usage() {
    echo "Usage:"
    echo "  Function 1 (Replace link): $0 set <target file>    Example: $0 set agent.v1.1.3"
    echo "  Function 2 (View version): $0 version"
    exit 1
}

# Function 1: Set link (replace link)
set_link() {
    local target_filename="$1"
    # Concatenate the complete target file path (prefix directory + file name)
    local target_file="${TARGET_DIR}/${target_filename}"

    # Check if the input parameter file exists (handle duplicate slashes in the path, e.g., ././agent.v1.0.3)
    #target_file=$(realpath -m "$target_file")

    # Check if the input parameter file exists
    if [ ! -f "$target_file" ]; then
        echo "Failed: File $target_file does not exist!"
        exit 1
    fi

    # Delete the original link (if it exists)
    if [ -L "$LINK_NAME" ]; then
        rm -f "$LINK_NAME"
        echo "Original link $LINK_NAME has been deleted"
    fi

    # Create new link (use absolute or relative path, keep consistent with the original file)
    ln -s "$target_file" "$LINK_NAME"
    if [ $? -eq 0 ]; then
        echo "Success: Link has been updated to $LINK_NAME -> $target_file"
        exit 0
    else
        echo "Failed: Failed to create link!"
        exit 1
    fi
}

# Function 2: View current link version
show_version() {
    # Check if the link exists
    if [ ! -L "$LINK_NAME" ]; then
        echo "There is no valid link $LINK_NAME currently"
        exit 1
    fi

    # Get the target file pointed to by the link (parse the complete path)
    target=$(readlink -f "$LINK_NAME")
    # Extract version number (match format v+number.number.number)
    version=$(echo "$target" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+')

    if [ -n "$version" ]; then
        echo "Current link version: $version"
        # Optional: Show complete path
        echo "Link points to complete path: $target"
    else
        echo "Current link points to: $target (Version number format not recognized)"
    fi
    exit 0
}

# Main logic: Parameter check and function distribution
if [ $# -eq 0 ]; then
    usage
fi

case "$1" in
    set)
        if [ $# -ne 2 ]; then
            echo "Error: set command requires specifying the target file!"
            usage
        fi
        set_link "$2"
        ;;
    version)
        show_version
        ;;
    *)
        echo "Error: Invalid command $1"
        usage
        ;;
esac