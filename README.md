[![Go Report Card](http://goreportcard.com/badge/mattbostock/rotate)](http://goreportcard.com/report/mattbostock/rotate)

# rotate

A small command-line utility to rotate a file or directory.

Follows a specified schedule passed as a commandline argument and purges old rotations.

## Usage

```
./rotate -help
Usage of ./rotate:
  -format="2006-01-02": date format used in filenames as a representation of January 2, 2006
  -schedule="1d:7,1w:4,1m:12,1y:4": rotation schedule and retention period
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