#!/bin/sh -e
# Checkout https://github.com/ahaoboy/js-engine-benchmark/tree/main/v8-v7

mkdir -p testdata/js-engine-benchmark
git clone --depth 1 --filter=blob:none --sparse https://github.com/ahaoboy/js-engine-benchmark.git testdata/js-engine-benchmark
cd testdata/js-engine-benchmark
git sparse-checkout set v8-v7
