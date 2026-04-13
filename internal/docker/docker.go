package docker

import (
	"fmt"
	"os/exec"
)

// Check verifies that Docker is installed and the daemon is running.
func Check() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("Docker is not installed. Install it from https://docs.docker.com/get-docker/")
	}

	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker is not running. Start Docker Desktop and try again")
	}

	return nil
}
