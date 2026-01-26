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
	"maps"
	"testing"

	dockerclient "github.com/moby/moby/client"
)

func TestNew(t *testing.T) {
	client := New()

	if client == nil {
		t.Fatal("Expected New() to return a non-nil client")
	}

	if client.containerCache == nil {
		t.Error("Expected containerCache to be initialized")
	}

	if len(client.containerCache) != 0 {
		t.Errorf("Expected containerCache to be empty, got %d items", len(client.containerCache))
	}
}

func TestGetContainerNameByID_WithoutDockerClient(t *testing.T) {
	client := &Client{
		containerCache: make(map[string]string),
		dockerClient:   nil,
	}

	result := client.GetContainerNameByID("test-container-id")
	if result != "" {
		t.Errorf("Expected empty string when dockerClient is nil, got %s", result)
	}
}

func TestGetContainerNameByID_FromCache(t *testing.T) {
	// Test that cache lookups work when entries exist
	// Note: We need dockerClient to be non-nil, otherwise GetContainerNameByID returns "" immediately
	// But we don't need it to be functional for cache hits
	client := &Client{
		containerCache: map[string]string{
			"abc123": "test-container",
			"def456": "another-container",
		},
		dockerClient: &dockerclient.Client{}, // Non-nil to allow cache lookups
	}

	// Test cache hits - these should return immediately without using dockerClient
	result := client.GetContainerNameByID("abc123")
	if result != "test-container" {
		t.Errorf("GetContainerNameByID(%q) = %q, expected %q", "abc123", result, "test-container")
	}

	result = client.GetContainerNameByID("def456")
	if result != "another-container" {
		t.Errorf(
			"GetContainerNameByID(%q) = %q, expected %q",
			"def456",
			result,
			"another-container",
		)
	}
}

func TestGetContainerNameByID_WithNilDockerClient(t *testing.T) {
	// When dockerClient is nil, GetContainerNameByID should always return empty string
	client := &Client{
		containerCache: map[string]string{
			"abc123": "test-container",
		},
		dockerClient: nil,
	}

	result := client.GetContainerNameByID("abc123")
	if result != "" {
		t.Errorf("Expected empty string when dockerClient is nil, got %q", result)
	}
}

func TestContainerCache_ConcurrentAccess(t *testing.T) {
	client := New()

	client.containerCache["container1"] = "name1"
	client.containerCache["container2"] = "name2"

	done := make(chan bool)

	for range 10 {
		go func() {
			for range 100 {
				_ = client.GetContainerNameByID("container1")
				_ = client.GetContainerNameByID("container2")
				_ = client.GetContainerNameByID("nonexistent")
			}

			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}

func TestClient_Initialization(t *testing.T) {
	client := New()

	if client.containerCache == nil {
		t.Error("containerCache should not be nil")
	}

	testID := "test-id-123"
	testName := "test-container-name"

	client.containerCacheMutex.Lock()
	client.containerCache[testID] = testName
	client.containerCacheMutex.Unlock()

	result := client.GetContainerNameByID(testID)
	if result != testName {
		t.Errorf("Expected to retrieve %q, got %q", testName, result)
	}
}

func TestRefreshContainerCache_WithoutDockerClient(t *testing.T) {
	client := &Client{
		containerCache: map[string]string{
			"old-id": "old-container",
		},
		dockerClient: nil,
	}

	client.refreshContainerCache()

	// When dockerClient is nil, refreshContainerCache() returns early without clearing cache
	if len(client.containerCache) != 1 {
		t.Errorf(
			"Expected cache to remain unchanged with 1 item, got %d items",
			len(client.containerCache),
		)
	}

	// Verify the original item is still there
	if name, exists := client.containerCache["old-id"]; !exists || name != "old-container" {
		t.Error("Expected original cache entry to remain unchanged")
	}
}

func TestClient_EmptyCache(t *testing.T) {
	client := New()

	result := client.GetContainerNameByID("any-id")
	if result != "" {
		t.Errorf("Expected empty string from empty cache, got %s", result)
	}
}

func TestClient_CachePopulation(t *testing.T) {
	client := New()

	testCases := map[string]string{
		"id1": "container-one",
		"id2": "container-two",
		"id3": "container-three",
	}

	client.containerCacheMutex.Lock()

	maps.Copy(client.containerCache, testCases)

	client.containerCacheMutex.Unlock()

	for id, expectedName := range testCases {
		result := client.GetContainerNameByID(id)
		if result != expectedName {
			t.Errorf("GetContainerNameByID(%q) = %q, expected %q", id, result, expectedName)
		}
	}
}
