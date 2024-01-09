//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/aserto-dev/mage-loot/common"
	"github.com/aserto-dev/mage-loot/deps"
	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"
)

func init() {
	os.Setenv("GO_VERSION", "1.20")
}

// Build builds all binaries in ./cmd.
func Build() error {
	return common.BuildReleaser()
}

// BuildAll builds all binaries in ./cmd for
// all configured operating systems and architectures.
func BuildAll() error {
	return common.BuildAllReleaser("--rm-dist", "--snapshot")
}

// Lint runs linting for the entire project.
func Lint() error {
	return common.Lint()
}

// Test runs all tests and generates a code coverage report.
func Test() error {
	return common.Test("-timeout", "240s", "-parallel=1")
}

func Deps() {
	deps.GetAllDeps()
}

// All runs all targets in the appropriate order.
// The targets are run in the following order:
// deps, lint, test, build
func All() error {
	mg.SerialDeps(Deps, Lint, Test, Build)
	return nil
}

// Release releases the project.
func Release() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is undefined")
	}

	if err := writeVersion(); err != nil {
		return err
	}

	return common.Release("--rm-dist")
}

func writeVersion() error {
	// get tag.
	version, err := exec.Command("git", "describe", "--tags").Output()
	if err != nil {
		return errors.Wrap(err, "failed to get current git tag")
	}

	file, err := os.Create("VERSION.txt")
	if err != nil {
		return errors.Wrap(err, "failed to create version file")
	}

	defer file.Close()

	if _, err := file.Write(version); err != nil {
		return errors.Wrap(err, "failed to write to version file")
	}

	return file.Sync()
}
