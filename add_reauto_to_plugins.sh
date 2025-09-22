#!/bin/bash

# This script checks and updates the plugin.cfg file.

FILE="plugin.cfg"
PKG_NAME="rehosts"
PKG_REPO="github.com/eli-github/"
LINE_TO_CHECK="$PKG_NAME:$PKG_REPO$PKG_NAME"
# LINE_TO_CHECK="rehosts:github.com/eli-github/rehosts"
LINE_TO_INSERT_BEFORE="hosts:hosts"

# ---
# 1. Check if the input file exists.
# ---
if [ ! -f "$FILE" ]; then
    echo "Error: The file '$FILE' was not found."
    exit 1
fi

# ---
# 2. Check if the target line already exists.
# ---
# Using grep -q for a quiet check (no output), its exit status tells us if a match was found.
if grep -qF "$LINE_TO_CHECK" "$FILE"; then
    echo "The line '$LINE_TO_CHECK' already exists in '$FILE'. No changes needed."
    exit 0
fi

# ---
# 3. If the line does not exist, insert it using sed.
# ---
echo "The line was not found. Inserting '$LINE_TO_CHECK' before '$LINE_TO_INSERT_BEFORE'..."

# -i: edit the file in-place
# /^hosts:hosts/i: find the line that begins with "hosts:hosts" and use the 'i' (insert) command
# The text to be inserted is on the following line
sed -i "/^$LINE_TO_INSERT_BEFORE/i $LINE_TO_CHECK" "$FILE"

# go get github.com/eli-github/rehosts

if [ $? -eq 0 ]; then
    echo "Successfully inserted the line."
else
    echo "An error occurred during the insertion. Check if the line '$LINE_TO_INSERT_BEFORE' exists in the file."
    exit 1
fi
