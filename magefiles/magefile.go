//go:build mage

package main

import (
	"fmt"
	"os"

	"github.com/aserto-dev/mage-loot/common"
	"github.com/aserto-dev/mage-loot/deps"
	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"
)

const containerImage string = "scim"

func init() {
	os.Setenv("GO_VERSION", "1.22")
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

// DockerImage builds the docker image for the project.
func DockerImage() error {
	err := BuildAll()
	if err != nil {
		return err
	}
	version, err := common.Version()
	if err != nil {
		return errors.Wrap(err, "failed to calculate version")
	}

	return common.DockerImage(fmt.Sprintf("%s:%s", containerImage, version))
}

// DockerPush builds the docker image using all tags specified by sver
// and pushes it to the specified registry
func DockerPush(registry, org string) error {
	tags, err := common.DockerTags(registry, fmt.Sprintf("%s/%s", org, containerImage))
	if err != nil {
		return err
	}

	version, err := common.Version()
	if err != nil {
		return errors.Wrap(err, "failed to calculate version")
	}

	for _, tag := range tags {
		common.UI.Normal().WithStringValue("tag", tag).Msg("pushing tag")
		err = common.DockerPush(
			fmt.Sprintf("%s:%s", containerImage, version),
			fmt.Sprintf("%s/%s/%s:%s", registry, org, containerImage, tag),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func Deps() {
	deps.GetAllDeps()
}

// All runs all targets in the appropriate order.
// The targets are run in the following order:
// deps, lint, test, build, dockerimage
func All() error {
	mg.SerialDeps(Deps, Lint, Test, Build, DockerImage)
	return nil
}

// Release releases the project.
func Release() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is undefined")
	}

	return common.Release("--rm-dist")
}
