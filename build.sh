#!/bin/bash

# 设置输出二进制文件名称
BINARY_NAME="pod_connect_exporter"

# 编译程序
echo "Building $BINARY_NAME..."
go build -o $BINARY_NAME ./cmd/exporter

if [ $? -eq 0 ]; then
    echo "Build successful! Binary: $BINARY_NAME"
else
    echo "Build failed!"
    exit 1
fi 