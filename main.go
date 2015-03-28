package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/termie/go-shutil"
)

const daysPerWeek = 7

var version string

type rotation struct {
	frequency frequency
	retention uint64
}

type frequency struct {
	label  string
	years  uint16
	months uint16
	days   uint16
}

func parseSchedule(s string) (schedule []*rotation, err error) {
	for _, v := range strings.Split(s, ",") {
		rotation := &rotation{}

		parts := strings.Split(v, ":")
		if len(parts) != 2 {
			return schedule, errors.New("Colon delimiter not found")
		}
		frequencyString := parts[0]
		retentionString := parts[1]

		var (
			freqUnit     string
			freqInterval uint16
		)

		_, err := fmt.Sscanf(frequencyString, "%d%s", &freqInterval, &freqUnit)
		if err != nil {
			return schedule, errors.New("Frequency must be an integer followed by 'd', 'w', 'm', or 'y'")
		}

		rotation.frequency.label = frequencyString

		switch freqUnit {
		case "d":
			rotation.frequency.days = freqInterval
		case "w":
			rotation.frequency.days = freqInterval * daysPerWeek
		case "m":
			rotation.frequency.months = freqInterval
		case "y":
			rotation.frequency.years = freqInterval
		default:
			return schedule, errors.New("Frequency must be an integer followed by 'd', 'w', 'm', or 'y'")
		}

		rotation.retention, err = strconv.ParseUint(retentionString, 10, 16)
		if err != nil {
			return schedule, errors.New("Number of rotations to retain must be specified as an integer")
		}

		schedule = append(schedule, rotation)
	}

	return schedule, nil
}

type lastRotation struct {
	fileInfo    os.FileInfo
	rotatedTime time.Time
}
type lastRotations []lastRotation

func (p lastRotations) Len() int {
	return len(p)
}

func (p lastRotations) Less(a, b int) bool {
	return p[a].rotatedTime.Before(p[b].rotatedTime)
}

func (p lastRotations) Swap(a, b int) {
	p[a], p[b] = p[b], p[a]
}

var config struct {
	format   string
	schedule string
	version  bool
	verbose  bool
}

func init() {
	flag.StringVar(&config.format, "format", "2006-01-02", "date format used in filenames as a representation of January 2, 2006")
	flag.StringVar(&config.schedule, "schedule", "1d:7,1w:4,1m:12,1y:4", "rotation schedule and retention period")
	flag.BoolVar(&config.version, "version", false, "prints current version")
	flag.BoolVar(&config.verbose, "verbose", false, "verbose output")
}

func main() {
	if version == "" {
		fmt.Println(makeFileNotUsed)
		os.Exit(1)
	}

	flag.Parse()

	if config.version {
		fmt.Println(version)
		os.Exit(0)
	}

	schedule, err := parseSchedule(config.schedule)
	if err != nil {
		fmt.Printf("Could not parse schedule: %s\n", err)
		flag.Usage()
		os.Exit(1)
	}

	var cwd string
	if cwd, err = getCwd(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var (
		source = cwd
		target = cwd
	)

	if len(flag.Args()) > 0 {
		source, err = filepath.Abs(flag.Args()[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if len(flag.Args()) > 1 {
		target, err = filepath.Abs(flag.Args()[1])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	err = rotate(schedule, source, target)

	if err != nil {
		fmt.Println(err)
		flag.Usage()
		os.Exit(1)
	}
}

func rotate(schedule []*rotation, source, target string) error {
	if source == target {
		return fmt.Errorf("Target path %q cannot be the same as the source path %q", target, source)
	}

	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("Source path %q does not exist", source)
	}

	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("Target path %q does not exist", target)
	}

	if dirInfo, _ := os.Stat(target); !dirInfo.IsDir() {
		return fmt.Errorf("Target path %q must be a directory", target)
	}

	for _, rotation := range schedule {
		var foundRotations lastRotations

		rotationDir := filepath.Join(target, rotation.frequency.label)
		os.Mkdir(rotationDir, 0755) // FIXME: Respect umask

		files, err := ioutil.ReadDir(rotationDir)
		if err != nil {
			return err
		}

		for _, f := range files {
			if !f.IsDir() {
				continue
			}

			var rotatedTime time.Time
			if rotatedTime, err = time.Parse(config.format, f.Name()); err != nil {
				fmt.Fprintf(os.Stderr, "Ignoring directory with mismatched date format: %q", filepath.Join(rotationDir, f.Name()))
				continue
			}

			foundRotations = append(foundRotations, lastRotation{f, rotatedTime})
		}

		sort.Sort(foundRotations)

		// Get current time to the resolution allowed by the user-specified date format
		timeNow, err := time.Parse(config.format, time.Now().Format(config.format))
		if err != nil {
			panic(err)
		}

		curDir := filepath.Join(rotationDir, timeNow.Format(config.format))
		rotateNow := false

		if len(foundRotations) > 0 {
			lastRotateTime := timeNow.AddDate(-int(rotation.frequency.years), -int(rotation.frequency.months), -int(rotation.frequency.days))
			if foundRotations[0].rotatedTime.Before(lastRotateTime) || foundRotations[0].rotatedTime.Equal(lastRotateTime) {
				rotateNow = true
			}
		} else {
			rotateNow = true
		}

		if rotateNow {
			var sourceInfo os.FileInfo
			if sourceInfo, err = os.Stat(source); err != nil {
				return err
			}

			if config.verbose {
				fmt.Printf("Copying from %q to %q\n", source, curDir)
			}

			if sourceInfo.IsDir() {
				if err = shutil.CopyTree(source, curDir, nil); err != nil {
					return err
				}
			} else {
				os.Mkdir(curDir, 0755) // FIXME: Respect umask
				if _, err = shutil.Copy(source, curDir, true); err != nil {
					return err
				}
			}
		}

		if rotation.retention > 0 {
			var retainCount = rotation.retention

			if rotateNow {
				retainCount--
			}

			var index = retainCount
			if retainCount > uint64(len(foundRotations)) {
				index = uint64(len(foundRotations))
			}

			toDelete := foundRotations[index:]

			var path string
			for _, d := range toDelete {
				path = filepath.Join(rotationDir, d.fileInfo.Name())

				if config.verbose {
					fmt.Printf("Deleting directory: %s\n", path)
				}

				os.RemoveAll(path)
			}

		}
	}

	return nil
}

func getCwd() (cwd string, err error) {
	cwd, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return cwd, err
	}

	return cwd, err
}

const makeFileNotUsed = "Makefile was not used when compiling binary, run 'make' to re-compile"
