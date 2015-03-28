package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func runCmd(t *testing.T, arg ...string) (int, string) {
	cmd := exec.Command("./rotate-test", arg...)
	output, err := cmd.CombinedOutput()

	var code int
	switch err.(type) {
	case *os.PathError:
		t.Fatal("You must run the tests using `make test`")
	case *exec.ExitError:
		code = cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
	case nil:
		// continue
	default:
		t.Fatalf("%#v", err)
	}

	return code, string(output)
}

func TestVersionFlag(t *testing.T) {
	exitcode, output := runCmd(t, "-version")

	assert.Equal(t, exitcode, 0)
	assert.Equal(t, output, "testversion\n")
}

func TestNoArgs(t *testing.T) {
	exitcode, output := runCmd(t)

	assert.Equal(t, exitcode, 1)
	assert.Contains(t, output, "cannot be the same as the source path")
}

func TestNonExistentSourceDir(t *testing.T) {
	exitcode, output := runCmd(t, "foo")

	assert.Equal(t, exitcode, 1)
	assert.Contains(t, output, "does not exist")
}

func TestNonExistentDestDir(t *testing.T) {
	os.Mkdir("foo", 0755)
	defer os.Remove("foo")

	exitcode, output := runCmd(t, "foo", "bar")

	assert.Equal(t, exitcode, 1)
	assert.Contains(t, output, "does not exist")
}

func TestInvalidScheduleValues(t *testing.T) {
	type example struct {
		frequencyInterval string
		frequencyUnit     string
		retention         string
		expectedError     string
	}

	examples := []example{
		{
			frequencyInterval: "z",
			frequencyUnit:     "d",
			retention:         "7",
			expectedError:     "Frequency must be an integer followed by 'd', 'w', 'm', or 'y'",
		},
		{
			frequencyInterval: "1",
			frequencyUnit:     "z",
			retention:         "7",
			expectedError:     "Frequency must be an integer followed by 'd', 'w', 'm', or 'y'",
		},
		{
			frequencyInterval: "1",
			frequencyUnit:     "d",
			retention:         "z",
			expectedError:     "Number of rotations to retain must be specified as an integer",
		},
	}

	for _, e := range examples {
		exitcode, output := runCmd(t, "-schedule="+e.frequencyInterval+e.frequencyUnit+":"+e.retention)

		assert.Equal(t, exitcode, 1)
		assert.Contains(t, output, e.expectedError)
	}
}

func TestInvalidScheduleDelimiter(t *testing.T) {
	exitcode, output := runCmd(t, "-schedule=1d:7;2w;4")

	assert.Equal(t, exitcode, 1)
	assert.Contains(t, output, "Number of rotations to retain must be specified as an integer")
}

func TestRotation(t *testing.T) {
	now := time.Now()

	type example struct {
		source   string
		dest     string
		contents []string
		rotated  []string
		expected []string
		schedule []string
	}

	examples := []example{
		// Happy path
		{
			source:   "foo",
			dest:     "bar",
			contents: []string{filepath.Join("baz", "fileA"), "fileB"},
			rotated:  []string{},
			expected: []string{
				"bar/1d/" + now.Format("2006-01-02") + "/baz/fileA",
				"bar/1d/" + now.Format("2006-01-02") + "/fileB",
				"bar/1m/" + now.Format("2006-01-02") + "/baz/fileA",
				"bar/1m/" + now.Format("2006-01-02") + "/fileB",
				"bar/1w/" + now.Format("2006-01-02") + "/baz/fileA",
				"bar/1w/" + now.Format("2006-01-02") + "/fileB",
				"bar/1y/" + now.Format("2006-01-02") + "/baz/fileA",
				"bar/1y/" + now.Format("2006-01-02") + "/fileB",
			},
			schedule: []string{"1d:7", "1m:12", "1w:4", "1y:4"},
		},
		// Existing rotations
		{
			source:   "foo",
			dest:     "bar",
			contents: []string{"fileA"},
			rotated: []string{
				filepath.Join("1d", now.AddDate(0, 0, -1).Format("2006-01-02")),
				filepath.Join("1w", now.AddDate(0, 0, -16).Format("2006-01-02")),
			},
			expected: []string{
				"bar/1d/" + now.AddDate(0, 0, -1).Format("2006-01-02") + "/fileA",
				"bar/1d/" + now.Format("2006-01-02") + "/fileA",
				"bar/1m/" + now.Format("2006-01-02") + "/fileA",
				"bar/1w/" + now.AddDate(0, 0, -16).Format("2006-01-02") + "/fileA",
				"bar/1w/" + now.Format("2006-01-02") + "/fileA",
				"bar/1y/" + now.Format("2006-01-02") + "/fileA",
			},
			schedule: []string{"1d:7", "1m:12", "1w:4", "1y:4"},
		},
		// Multiple rotations exist for a given frequency, i.e. two monthly rotations exist
		// Rotations exist for "2y" frequency but not old enough to warrant a new rotation
		{
			source:   "foo",
			dest:     "bar",
			contents: []string{"fileA"},
			rotated: []string{
				filepath.Join("2y", now.AddDate(-1, 0, 0).Format("2006-01-02")),
				filepath.Join("1m", now.AddDate(0, -2, 0).Format("2006-01-02")),
				filepath.Join("3m", now.AddDate(0, -1, 0).Format("2006-01-02")),
			},
			expected: []string{
				"bar/1m/" + now.AddDate(0, -2, 0).Format("2006-01-02") + "/fileA",
				"bar/1m/" + now.Format("2006-01-02") + "/fileA",
				"bar/2y/" + now.AddDate(-1, 0, 0).Format("2006-01-02") + "/fileA",
				"bar/3m/" + now.AddDate(0, -1, 0).Format("2006-01-02") + "/fileA",
				"bar/5w/" + now.Format("2006-01-02") + "/fileA",
			},
			schedule: []string{"1m:10", "3m:10", "5w:10", "2y:10"},
		},
		// Old rotations are purged according to the retention policy set
		{
			source:   "foo",
			dest:     "bar",
			contents: []string{"fileA"},
			rotated: []string{
				filepath.Join("1d", now.AddDate(0, 0, -2).Format("2006-01-02")),
				filepath.Join("1d", now.AddDate(0, 0, -1).Format("2006-01-02")),
				filepath.Join("1w", now.AddDate(0, 0, -16).Format("2006-01-02")),
			},
			expected: []string{
				"bar/1d/" + now.Format("2006-01-02") + "/fileA",
				"bar/1w/" + now.Format("2006-01-02") + "/fileA",
			},
			schedule: []string{"1d:1", "1w:1"},
		},
	}

	for _, e := range examples {
		os.Mkdir(e.source, 0755)
		os.Mkdir(e.dest, 0755)

		createFixtures := func(path string) {
			for _, f := range e.contents {
				dir, _ := filepath.Split(f)
				os.MkdirAll(filepath.Join(path, dir), 0755)

				_, err := os.Create(filepath.Join(path, f))
				if err != nil {
					panic(err)
				}
			}
		}

		createFixtures(e.source)

		for _, r := range e.rotated {
			createFixtures(filepath.Join(e.dest, r))
		}

		exitcode, _ := runCmd(t, "-schedule="+strings.Join(e.schedule, ","), e.source, e.dest)

		assert.True(
			t,
			reflect.DeepEqual(e.expected, dirTree(e.dest)),
			fmt.Sprintf("File structure does not match expected:\nExpected:\n%#v\n\nGot:\n%#v\n", e.expected, dirTree(e.dest)),
		)
		assert.Equal(t, exitcode, 0)

		os.RemoveAll(e.source)
		os.RemoveAll(e.dest)

	}
}

func dirTree(dir string) []string {
	f, err := os.Open(dir)
	if err != nil {
		panic(err)
	}

	names, err := f.Readdirnames(0)
	if err != nil {
		panic(err)
	}

	var ret []string
	for _, d := range names {
		contents := dirTree(filepath.Join(dir, d))
		if len(contents) == 0 {
			ret = append(ret, filepath.Join(dir, d))
		} else {
			ret = append(ret, contents...)
		}
	}

	return ret

}
