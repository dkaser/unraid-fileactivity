package docker

/*
	fileactivity-watcher
	Copyright (C) 2025-2026 Derek Kaser

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

type Client struct {
	containerCache      map[string]string
	containerCacheMutex sync.RWMutex
	dockerClient        *client.Client
}

func New() *Client {
	containerCache := make(map[string]string)

	// Initialize Docker client
	var err error

	dockerClient, err := client.New()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create Docker client")
	}

	return &Client{
		containerCache: containerCache,
		dockerClient:   dockerClient,
	}
}

func (c *Client) GetContainerNameByID(containerID string, ctx context.Context) string {
	if c.dockerClient == nil {
		return ""
	}

	// Check cache first (read lock)
	c.containerCacheMutex.RLock()

	if name, exists := c.containerCache[containerID]; exists {
		c.containerCacheMutex.RUnlock()

		return name
	}

	c.containerCacheMutex.RUnlock()

	// Not in cache, refresh container list
	c.refreshContainerCache(ctx)

	// Check cache again after refresh (read lock)
	c.containerCacheMutex.RLock()
	defer c.containerCacheMutex.RUnlock()

	if name, exists := c.containerCache[containerID]; exists {
		return name
	}

	return ""
}

func (c *Client) refreshContainerCache(ctx context.Context) {
	if c.dockerClient == nil {
		return
	}

	// Create new cache
	newCache := make(map[string]string)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.dockerClient.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to list Docker containers")

		return
	}

	for _, ctr := range result.Items {
		if len(ctr.Names) == 0 {
			continue
		}

		newCache[ctr.ID] = strings.TrimPrefix(ctr.Names[0], "/")
	}

	c.containerCacheMutex.Lock()
	c.containerCache = newCache
	c.containerCacheMutex.Unlock()

	log.Debug().Int("cached_containers", len(newCache)).Msg("Refreshed container cache")
}
