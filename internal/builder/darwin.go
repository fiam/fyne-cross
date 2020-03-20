package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lucor/fyne-cross/internal/volume"
)

// NewDarwin returns a builder for the Darwin OS
func NewDarwin(opts Options) *Darwin {
	return &Darwin{
		os:   "darwin",
		opts: opts,
	}
}

// Darwin is the build for the Darwin OS
type Darwin struct {
	os   string
	opts Options
}

// PreBuild performs all tasks needed to perform a build
func (b *Darwin) PreBuild(vol *volume.Volume, opts PreBuildOptions) error {
	//ensures go.mod exists, if not try to create a temporary one
	return goModInit(b, vol, opts.Verbose)
}

// Build builds the package
func (b *Darwin) Build(vol *volume.Volume, opts BuildOptions) error {

	output := filepath.Join(vol.BinDirContainer(), b.TargetID(), b.Output())

	// add default ldflags, if any
	if ldflags := b.BuildLdFlags(); ldflags != nil {
		opts.LdFlags = append(opts.LdFlags, ldflags...)
	}

	// add default tags, if any
	if tags := b.BuildTags(); tags != nil {
		opts.Tags = append(opts.Tags, tags...)
	}

	command := goBuildCmd(output, opts)
	err := runBuilderDockerCmd(b, vol, b.BuildEnv(), vol.WorkDirContainer(), command, opts.Verbose)
	if err != nil {
		return fmt.Errorf("Could not build for %s/%s: %v", b.os, b.opts.Arch, err)
	}

	return nil
}

//BuildEnv returns the env variables required to build the package
func (b *Darwin) BuildEnv() []string {
	switch b.opts.Arch {
	case "amd64":
		return []string{"GOOS=darwin", "GOARCH=amd64", "CC=o32-clang"}
	case "386":
		return []string{"GOOS=darwin", "GOARCH=386", "CC=o32-clang"}
	}
	return []string{}
}

//BuildLdFlags returns the default ldflags used to build the package
func (b *Darwin) BuildLdFlags() []string {
	return nil
}

//BuildTags returns the default tags used to build the package
func (b *Darwin) BuildTags() []string {
	return nil
}

// DockerImage returns the Docker image name used for building
func (b *Darwin) DockerImage() string {
	if b.opts.DockerImage != "" {
		return b.opts.DockerImage
	}
	return darwinDockerImage
}

// TargetID returns the target ID for the builder
func (b *Darwin) TargetID() string {
	return fmt.Sprintf("%s-%s", b.os, b.opts.Arch)
}

// Output returns the named output
func (b *Darwin) Output() string {
	return b.opts.Output
}

// Package generate a package for distribution
func (b *Darwin) Package(vol *volume.Volume, opts PackageOptions) error {
	// copy the icon to tmp dir
	err := cp(opts.Icon, filepath.Join(vol.TmpDirHost(), defaultIcon))
	if err != nil {
		return fmt.Errorf("Could not package the Fyne app due to error copying the icon: %v", err)
	}

	// use the fyne package command to create the dist package
	packageName := fmt.Sprintf("%s.app", b.Output())
	command := []string{
		fyneCmd, "package",
		"-os", b.os,
		"-executable", filepath.Join(vol.BinDirContainer(), b.TargetID(), b.Output()),
		"-name", b.Output(),
	}
	// set appID if specified
	if opts.AppID != "" {
		command = append(command, "-appID", opts.AppID)
	}

	err = runBuilderDockerCmd(b, vol, []string{}, vol.TmpDirContainer(), command, opts.Verbose)
	if err != nil {
		return fmt.Errorf("Could not package the Fyne app: %v", err)
	}

	// move the dist package into the "dist" folder
	srcFile := filepath.Join(vol.TmpDirHost(), packageName)
	distFile := filepath.Join(vol.DistDirHost(), b.TargetID(), packageName)
	err = os.MkdirAll(filepath.Dir(distFile), 0755)
	if err != nil {
		return fmt.Errorf("Could not create the dist package dir: %v", err)
	}
	// Remove previous build to avoid rename to fail, if any
	os.RemoveAll(distFile)
	return os.Rename(srcFile, distFile)
}
