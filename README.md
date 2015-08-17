[![Go Report Card](http://goreportcard.com/badge/mattbostock/rotate)](http://goreportcard.com/report/mattbostock/rotate)
[![Build Status](https://travis-ci.org/mattbostock/rotate.svg?branch=master)](https://travis-ci.org/mattbostock/rotate)

# rotate

A small command-line utility to rotate a file or directory.

Follows a specified schedule passed as a commandline argument and purges old rotations.

## Rationale

I've seen a number scripts written to perform backups implement file rotation
in diverse ways, often without tests to verify their behaviour.

To avoid reinventing the wheel each time, it occurred to me that it would be
useful to have a utility that does the one job of rotating a file or directory.
I couldn't find any existing utility that compiles to a binary to fulfill that
task, so I wrote this one.

## Usage

```
./rotate -help
Usage of ./rotate:
  -schedule="1d:7,1w:5,1m:12,1y:4": rotation schedule and retention period
  -verbose=false: verbose output
  -version=false: prints current version
```

## Example using default schedule
This example shows the first rotation of a directory named `source` using the default schedule and retention policy.

```
$ tree source
source
├── fileA
├── fileB
├── dirA
└── dirB
$ rotate source target
$ tree target
target
├── 1d
│   └── 2015-08-15
│       ├── fileA
│       ├── fileB
│       ├── dirA
│       └── dirB
├── 1m
│   └── 2015-08-15
│       ├── fileA
│       ├── fileB
│       ├── dirA
│       └── dirB
├── 1w
│   └── 2015-08-15
│       ├── fileA
│       ├── fileB
│       ├── dirA
│       └── dirB
└── 1y
    └── 2015-08-15
        ├── fileA
        ├── fileB
        ├── dirA
        └── dirB
```
