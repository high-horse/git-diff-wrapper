package git

import (
	"bytes"
	"os/exec"
)

func GetDiff(cached bool)(string, error) {
	args := []string{"diff", "--unified=3"}
	if cached {
		args = append(args, "--cached")
	}
	
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}