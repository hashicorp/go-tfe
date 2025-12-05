#!/usr/bin/env bash

# Copyright IBM Corp. 2018, 2025
# SPDX-License-Identifier: MPL-2.0

if ! gofmt -l -s .; then
    echo "gofmt found some files that need to be formatted. You can use the command: \`make fmt\` to reformat code."
    exit 1
fi

exit 0
