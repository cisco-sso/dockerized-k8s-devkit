// Copyright © 2018 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kdk

import (
	"context"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/cisco-sso/kdk/internal/pkg/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/manifoldco/promptui"
)

func Prune(ctx context.Context, dockerClient *client.Client, logger logrus.Entry) error {
	logger.Info("Starting Prune...")

	var (
		imageIds                 []string
		runningContainerImageIds []string
		staleImageIds            []string
	)

	// Get containers
	containers, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		logger.WithField("error", err).Fatal("Failed to list docker containers")
	}
	// Get images
	images, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		logger.WithField("error", err).Fatal("Failed to list docker images")
	}

	// Iterate through containers and track running container imageIds
	for _, container := range containers {
		if strings.Contains(container.Status, "Up") {
			runningContainerImageIds = append(runningContainerImageIds, container.ImageID)
		}
	}

	// Iterate through images and track images that have a `kdk` label key
	for _, image := range images {
		for key := range image.Labels {
			if key == "kdk" {
				imageIds = append(imageIds, image.ID)
				break
			}
		}
	}

	// iterate through imageIds and add imageIds that are NOT associated with currently running containers
	for imageId := range imageIds {
		if utils.SliceContains(runningContainerImageIds, imageIds[imageId]) {
		} else {
			staleImageIds = append(staleImageIds, imageIds[imageId])
		}
	}

	if len(staleImageIds) > 0 {
		// iterate through staleImageIds, prompt user to confirm deletion
		for staleImage := range staleImageIds {
			targetImage := staleImageIds[staleImage]
			logger.Infof("Delete stale KDK image [%s]?", targetImage)
			prompt := promptui.Prompt{
				Label:     "Continue",
				IsConfirm: true,
			}
			if _, err := prompt.Run(); err != nil {
				logger.Error("KDK stale image deletion canceled or invalid input.")
				return err
			}
			if _, err := dockerClient.ImageRemove(ctx, targetImage, types.ImageRemoveOptions{Force: true, PruneChildren: true}); err != nil {
				logger.WithField("error", err).Fatalf("Failed to prune KDK image [%s]", targetImage)
				return err
			} else {
				logger.Infof("Deleted stale KDK image [%s]", targetImage)
			}
		}
	} else {
		logger.Infof("No stale KDK images to delete")
	}
	return nil
}
