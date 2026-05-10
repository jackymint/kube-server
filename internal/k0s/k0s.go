package k0s

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type K0s struct {
	nodeName string
	dataDir  string
}

func New(nodeName, dataDir string) *K0s {
	return &K0s{nodeName: nodeName, dataDir: dataDir}
}

func (k *K0s) ssh(cmd string) (string, error) {
	sock := filepath.Join(k.dataDir, "nodes", k.nodeName, "vfkit.sock")
	out, err := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", fmt.Sprintf("ProxyCommand=socat - UNIX-CLIENT:%s", sock),
		"root@localhost", cmd,
	).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (k *K0s) InstallController() error {
	cmds := []string{
		"apk add --no-cache curl",
		"curl -sSLf https://get.k0s.sh | sh",
		"k0s install controller --single",
		"k0s start",
	}
	for _, cmd := range cmds {
		if out, err := k.sshRun(cmd); err != nil {
			return fmt.Errorf("cmd %q: %s: %w", cmd, out, err)
		}
	}
	return nil
}

func (k *K0s) InstallWorker(token string) error {
	cmds := []string{
		"apk add --no-cache curl",
		"curl -sSLf https://get.k0s.sh | sh",
		fmt.Sprintf("k0s install worker --token-file <(echo %s)", token),
		"k0s start",
	}
	for _, cmd := range cmds {
		if out, err := k.sshRun(cmd); err != nil {
			return fmt.Errorf("cmd %q: %s: %w", cmd, out, err)
		}
	}
	return nil
}

func (k *K0s) GetJoinToken() (string, error) {
	return k.ssh("k0s token create --role=worker")
}

func (k *K0s) GetKubeconfig() (string, error) {
	return k.ssh("k0s kubeconfig admin")
}

func (k *K0s) sshRun(cmd string) (string, error) {
	return k.ssh(cmd)
}
