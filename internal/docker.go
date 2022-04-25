package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Pull the docker image and tag as tag
// Return the digest
func dockerPullAndTag(ctx context.Context, identifier, tag string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"docker", "pull", identifier,
	)
	if err := runCommand(cmd); err != nil {
		return "", err
	}

	cmd = exec.CommandContext(ctx, "docker", "image", "inspect", identifier)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("command %s failed: %s", cmd, err)
	}
	var inspectJson []struct {
		RepoDigests []string
	}
	if err := json.Unmarshal(out, &inspectJson); err != nil {
		return "", fmt.Errorf("cannot parse output of %s: %s", cmd, err)
	}
	if len(inspectJson) != 1 {
		return "", fmt.Errorf("expected 1 item from %s, got %d", cmd, len(inspectJson))
	}
	if len(inspectJson[0].RepoDigests) < 1 {
		return "", fmt.Errorf("failed to get digest of %s", identifier)
	}
	digest := inspectJson[0].RepoDigests[0]

	cmd = exec.CommandContext(ctx,
		"docker", "tag", digest, tag,
	)
	if err := runCommand(cmd); err != nil {
		return "", err
	}

	return digest, nil
}
