#!/usr/bin/env sh

godog $@
chmod a+rw wait-for_coverage.out
cp wait-for_coverage.out /working
