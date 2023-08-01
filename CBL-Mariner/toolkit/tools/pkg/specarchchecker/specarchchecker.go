// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

// A tool for filtering specs by architecture.

package specarchchecker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/rpm"
	simpletoolchroot "github.com/microsoft/CBL-Mariner/toolkit/tools/pkg/simpletoolchroot"
)

type ArchChecker struct {
	simpleToolChroot simpletoolchroot.SimpleToolChroot
}

// New creates an ArchChecker. If the chroot is created successfully, the caller is responsible for calling CleanUp().
func New(buildDirPath, workerTarPath, specsDirPath string) (newArchChecker *ArchChecker, err error) {
	const chrootName = "specarchchecker_chroot"
	newArchChecker = &ArchChecker{}
	err = newArchChecker.simpleToolChroot.InitializeChroot(buildDirPath, chrootName, workerTarPath, specsDirPath)

	return newArchChecker, err
}

// CleanUp tears down the chroot
func (a *ArchChecker) CleanUp() error {
	return a.simpleToolChroot.CleanUp()
}

// FilterSpecsByArch converts a list of spec names to those that are compatible with the current architecture. Will create
// and destroy a chroot environment in the process.
func (a *ArchChecker) FilterSpecsByArch(specFiles []string, distTag string) (validSpecs []string, err error) {
	err = a.simpleToolChroot.RunInChroot(func() error {
		var runErr error
		validSpecs, runErr = a.filterListInChroot(specFiles, distTag)
		return runErr
	})
	if err != nil {
		return
	}

	return
}

// buildAllSpecsListFromNames builds a list of all spec file paths from a list of spec names. Paths are relative to the SPECS
// directory in the chroot.
func (a *ArchChecker) buildAllSpecsListFromNames(specNames []string) (specPaths []string, err error) {
	for _, specName := range specNames {
		var fullSpecPath []string
		specFilesGlob := filepath.Join(a.simpleToolChroot.ChrootRelativeSpecDir(), "**", fmt.Sprintf("%s.spec", specName))

		fullSpecPath, err = filepath.Glob(specFilesGlob)
		if err != nil {
			err = fmt.Errorf("failed while trying to enumerate all spec files with (%s). Error:\n%w", specFilesGlob, err)
			return
		}
		if len(fullSpecPath) != 1 {
			err = fmt.Errorf("expected to find exactly one spec file with (%s). Found %d", specFilesGlob, len(fullSpecPath))
			return
		}

		specPaths = append(specPaths, fullSpecPath[0])
	}

	return
}

func (a *ArchChecker) filterListInChroot(specFileNames []string, distTag string) (filteredSpecNames []string, err error) {
	const runChecks = false

	defines := rpm.DefaultDefinesWithDist(runChecks, distTag)
	specPaths, err := a.buildAllSpecsListFromNames(specFileNames)
	if err != nil {
		err = fmt.Errorf("failed to translate names to specs inside (%s). Error:\n%w", a.simpleToolChroot.ChrootRelativeSpecDir(), err)
		return
	}
	logger.Log.Debugf("Got specs: %v.", specPaths)
	filteredSpecs, err := rpm.BuildCompatibleSpecsList(a.simpleToolChroot.ChrootRelativeSpecDir(), specPaths, defines)
	if err != nil {
		err = fmt.Errorf("failed to retrieve a list of compatible  specs inside (%s). Error:\n%w", a.simpleToolChroot.ChrootRelativeSpecDir(), err)
		return
	}
	logger.Log.Debugf("Got filtered specs: %v.", filteredSpecs)

	for _, filteredSpec := range filteredSpecs {
		// We only want the base name of the spec file, without the .spec extension.
		specName := filepath.Base(filteredSpec)
		specName = strings.TrimSuffix(specName, ".spec")
		filteredSpecNames = append(filteredSpecNames, specName)
	}

	return
}
