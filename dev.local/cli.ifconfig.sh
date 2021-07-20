#!/bin/bash

ips=`ifconfig | grep "inet " | grep 198.18.0 | wc -l`

#fix docker network issue for Mac OS X platform
if [ "$(uname)" == "Darwin" ] && [ $ips != 31 ]
then
    # 0dns
    sudo ifconfig lo0 alias 198.18.0.98
    # sharders
    sudo ifconfig lo0 alias 198.18.0.81
    sudo ifconfig lo0 alias 198.18.0.82
    sudo ifconfig lo0 alias 198.18.0.83
    sudo ifconfig lo0 alias 198.18.0.84
    sudo ifconfig lo0 alias 198.18.0.85
    sudo ifconfig lo0 alias 198.18.0.86
    sudo ifconfig lo0 alias 198.18.0.87
    sudo ifconfig lo0 alias 198.18.0.88
    # miners
    sudo ifconfig lo0 alias 198.18.0.71
    sudo ifconfig lo0 alias 198.18.0.72
    sudo ifconfig lo0 alias 198.18.0.73
    sudo ifconfig lo0 alias 198.18.0.74
    sudo ifconfig lo0 alias 198.18.0.75
    sudo ifconfig lo0 alias 198.18.0.76
    sudo ifconfig lo0 alias 198.18.0.77
    sudo ifconfig lo0 alias 198.18.0.78
    # blobbers
    sudo ifconfig lo0 alias 198.18.0.91
    sudo ifconfig lo0 alias 198.18.0.92
    sudo ifconfig lo0 alias 198.18.0.93
    sudo ifconfig lo0 alias 198.18.0.94
    sudo ifconfig lo0 alias 198.18.0.95
    sudo ifconfig lo0 alias 198.18.0.96
    sudo ifconfig lo0 alias 198.18.0.97
    # validators
    sudo ifconfig lo0 alias 198.18.0.61
    sudo ifconfig lo0 alias 198.18.0.62
    sudo ifconfig lo0 alias 198.18.0.63
    sudo ifconfig lo0 alias 198.18.0.64
    sudo ifconfig lo0 alias 198.18.0.65
    sudo ifconfig lo0 alias 198.18.0.66
    sudo ifconfig lo0 alias 198.18.0.67
fi
