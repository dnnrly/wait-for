#!/usr/bin/env sh

set -o xtrace

godog $@
chmod a+rw wait-for_coverage.txt
cp wait-for_coverage.txt /working
