package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type VMConfig struct {
	Name    string
	CPU     int
	Memory  string
	Disk    string
	DataDir string
}

type VM struct {
	cfg     VMConfig
	cmd     *exec.Cmd
	pidFile string
}

func New(cfg VMConfig) *VM {
	return &VM{
		cfg:     cfg,
		pidFile: filepath.Join(cfg.DataDir, cfg.Name, "vm.pid"),
	}
}

func (v *VM) dir() string {
	return filepath.Join(v.cfg.DataDir, v.cfg.Name)
}

func (v *VM) diskPath() string {
	return filepath.Join(v.dir(), "disk.img")
}

func (v *VM) kernelPath() string {
	return filepath.Join(v.dir(), "vmlinuz")
}

func (v *VM) initrdPath() string {
	return filepath.Join(v.dir(), "initrd")
}

func memoryMB(mem string) string {
	mem = strings.ToUpper(mem)
	if strings.HasSuffix(mem, "GB") {
		n, _ := strconv.Atoi(strings.TrimSuffix(mem, "GB"))
		return strconv.Itoa(n * 1024)
	}
	return strings.TrimSuffix(mem, "MB")
}

func diskGB(disk string) int {
	disk = strings.ToUpper(disk)
	if strings.HasSuffix(disk, "GB") {
		n, _ := strconv.Atoi(strings.TrimSuffix(disk, "GB"))
		return n
	}
	return 10
}

func (v *VM) Create() error {
	if err := os.MkdirAll(v.dir(), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(v.diskPath()); os.IsNotExist(err) {
		size := fmt.Sprintf("%dg", diskGB(v.cfg.Disk))
		cmd := exec.Command("qemu-img", "create", "-f", "raw", v.diskPath(), size)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("create disk: %s: %w", out, err)
		}
	}
	return nil
}

func (v *VM) Start() error {
	if v.IsRunning() {
		return fmt.Errorf("vm %s already running", v.cfg.Name)
	}

	args := []string{
		"--cpus", strconv.Itoa(v.cfg.CPU),
		"--memory", memoryMB(v.cfg.Memory),
		"--kernel", v.kernelPath(),
		"--initrd", v.initrdPath(),
		"--kernel-cmdline", "console=hvc0 root=/dev/vda rw",
		"--device", fmt.Sprintf("virtio-blk,path=%s", v.diskPath()),
		"--device", "virtio-net,unixSocketPath=/var/run/socket_vmnet.sock",
		"--device", "virtio-serial",
		"--restful-uri", fmt.Sprintf("unix://%s", filepath.Join(v.dir(), "vfkit.sock")),
	}

	v.cmd = exec.Command("vfkit", args...)
	v.cmd.Stdout = os.Stdout
	v.cmd.Stderr = os.Stderr

	if err := v.cmd.Start(); err != nil {
		return fmt.Errorf("start vm: %w", err)
	}

	return os.WriteFile(v.pidFile, []byte(strconv.Itoa(v.cmd.Process.Pid)), 0644)
}

func (v *VM) Stop() error {
	pid, err := v.getPID()
	if err != nil {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		proc.Kill()
	}
	return os.Remove(v.pidFile)
}

func (v *VM) IsRunning() bool {
	pid, err := v.getPID()
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(os.Signal(nil)) == nil
}

func (v *VM) getPID() (int, error) {
	data, err := os.ReadFile(v.pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func (v *VM) ResizeDisk(newSize string) error {
	size := fmt.Sprintf("%dG", diskGB(newSize))
	cmd := exec.Command("qemu-img", "resize", v.diskPath(), size)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("resize disk: %s: %w", out, err)
	}
	return nil
}
