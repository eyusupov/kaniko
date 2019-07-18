/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kaniko/pkg/dockerfile"

	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/sirupsen/logrus"
)

type WorkdirCommand struct {
	BaseCommand
	cmd           *instructions.WorkdirCommand
	snapshotFiles []string
}

// For testing
var mkdir = os.MkdirAll

func updateWorkdir(workdirPath string, config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	replacementEnvs := buildArgs.ReplacementEnvs(config.Env)
	resolvedWorkingDir, err := util.ResolveEnvironmentReplacement(workdirPath, replacementEnvs, true)
	if err != nil {
		return err
	}
	if filepath.IsAbs(resolvedWorkingDir) {
		config.WorkingDir = resolvedWorkingDir
	} else {
		config.WorkingDir = filepath.Join(config.WorkingDir, resolvedWorkingDir)
	}
	logrus.Infof("Changed working directory to %s", config.WorkingDir)
	return nil
}

func (w *WorkdirCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: workdir")

	workdirPath := w.cmd.Path
	err := updateWorkdir(workdirPath, config, buildArgs)
	if err != nil {
		return err
	}

	// Only create and snapshot the dir if it didn't exist already
	w.snapshotFiles = []string{}
	if _, err := os.Stat(config.WorkingDir); os.IsNotExist(err) {
		logrus.Infof("Creating directory %s", config.WorkingDir)
		w.snapshotFiles = append(w.snapshotFiles, config.WorkingDir)
		return mkdir(config.WorkingDir, 0755)
	} else {
		// Cache the empty layer so we don't have to unpack FS on rerun
		return nil
	}
	return nil
}

// FilesToSnapshot returns the workingdir, which should have been created if it didn't already exist
func (w *WorkdirCommand) FilesToSnapshot() []string {
	return w.snapshotFiles
}

// String returns some information about the command for the image config history
func (w *WorkdirCommand) String() string {
	return w.cmd.String()
}

func (w *WorkdirCommand) MetadataOnly() bool {
	return false
}

func (r *WorkdirCommand) RequiresUnpackedFS() bool {
	return true
}

func (r *WorkdirCommand) ShouldCacheOutput() bool {
	return true
}

func (r *CachedWorkdirCommand) CacheCommand() DockerCommand {
	return &CachedWorkdirCommand{cmd: r.cmd}
}

type CachedWorkdirCommand struct {
	BaseCommand
	cmd *instructions.WorkdirCommand
}

func (cw *CachedWorkdirCommand) ExecuteCommand(config *v1.Config, buildArgs *dockerfile.BuildArgs) error {
	logrus.Info("cmd: workdir")

	return updateWorkdir(cw.cmd.Path, config, buildArgs)
}

func (cw *CachedWorkdirCommand) MetadataOnly() bool {
	return true
}

func (cw *CachedWorkdirCommand) String() string {
	return cw.cmd.String()
}
