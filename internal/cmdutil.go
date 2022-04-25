package internal

import (
	"log"
	"os"
	"os/exec"
)

func runCommand(cmd *exec.Cmd) error {
	log.Println("running", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Stderr.Write(out)
		log.Println("command failed:", cmd)
		return err
	}
	return nil
}
