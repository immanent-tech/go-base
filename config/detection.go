package config

import (
	"os"
	"strings"
)

// ContainerRuntime identifies the detected container environment.
type ContainerRuntime string

const (
	RuntimeNone     ContainerRuntime = "none"
	RuntimeDocker   ContainerRuntime = "docker"
	RuntimePodman   ContainerRuntime = "podman"
	RuntimeCloudRun ContainerRuntime = "cloudrun"
	RuntimeUnknown  ContainerRuntime = "unknown-container"
)

// DetectContainerRuntime tries to identify the container runtime we're
// executing under, checking the most specific/reliable signals first.
func DetectContainerRuntime() ContainerRuntime {
	// Cloud Run sets these unconditionally on every revision.
	// https://cloud.google.com/run/docs/container-contract#env-vars
	if os.Getenv("K_SERVICE") != "" && os.Getenv("K_REVISION") != "" {
		return RuntimeCloudRun
	}

	// Podman writes this file into every container it starts.
	// It's podman-specific — Docker does not create it.
	if fileExists("/run/.containerenv") {
		return RuntimePodman
	}

	// Podman (and some other OCI tools) also set this env var directly.
	if v := os.Getenv("container"); v == "podman" {
		return RuntimePodman
	}

	// Docker creates this empty file at container root.
	if fileExists("/.dockerenv") {
		return RuntimeDocker
	}

	// Fallback: inspect cgroup info (Linux only). Works for many
	// runtimes even when the marker files above are absent/masked.
	if rt := detectFromCgroup(); rt != RuntimeNone {
		return rt
	}

	return RuntimeNone
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func detectFromCgroup() ContainerRuntime {
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return RuntimeNone
	}
	content := string(data)

	switch {
	case strings.Contains(content, "libpod"):
		return RuntimePodman
	case strings.Contains(content, "docker"):
		return RuntimeDocker
	case strings.Contains(content, "kubepods"):
		return RuntimeUnknown // in a k8s pod, but runtime under that is ambiguous
	}
	return RuntimeNone
}
