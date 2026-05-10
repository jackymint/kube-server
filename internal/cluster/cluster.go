package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kube-server/kube-server/internal/config"
	"github.com/kube-server/kube-server/internal/k0s"
	"github.com/kube-server/kube-server/internal/vm"
)

type NodeRole string

const (
	RoleControlPlane NodeRole = "control-plane"
	RoleWorker       NodeRole = "worker"
)

type Node struct {
	Name string   `json:"name"`
	Role NodeRole `json:"role"`
	IP   string   `json:"ip"`
}

type Cluster struct {
	Name    string  `json:"name"`
	Nodes   []Node  `json:"nodes"`
	DataDir string  `json:"data_dir"`
	cfg     *config.Config
}

func New(name string, cfg *config.Config) *Cluster {
	return &Cluster{
		Name:    name,
		DataDir: filepath.Join(cfg.DataDir, "clusters", name),
		cfg:     cfg,
	}
}

func (c *Cluster) Create() error {
	if err := os.MkdirAll(c.DataDir, 0755); err != nil {
		return err
	}

	// control-plane node
	cpNode := Node{Name: fmt.Sprintf("%s-cp", c.Name), Role: RoleControlPlane}
	c.Nodes = append(c.Nodes, cpNode)

	// worker nodes
	for i := 1; i <= c.cfg.Cluster.WorkerCount; i++ {
		c.Nodes = append(c.Nodes, Node{
			Name: fmt.Sprintf("%s-worker-%d", c.Name, i),
			Role: RoleWorker,
		})
	}

	for _, node := range c.Nodes {
		nodeCfg := c.nodeVMConfig(node)
		v := vm.New(nodeCfg)
		if err := v.Create(); err != nil {
			return fmt.Errorf("create vm %s: %w", node.Name, err)
		}
	}

	return c.save()
}

func (c *Cluster) Start() error {
	if err := c.load(); err != nil {
		return err
	}

	// start control-plane first
	for _, node := range c.Nodes {
		if node.Role != RoleControlPlane {
			continue
		}
		v := vm.New(c.nodeVMConfig(node))
		if err := v.Start(); err != nil {
			return fmt.Errorf("start %s: %w", node.Name, err)
		}
		k := k0s.New(node.Name, c.DataDir)
		if err := k.InstallController(); err != nil {
			return fmt.Errorf("install k0s controller: %w", err)
		}
	}

	// get join token
	k := k0s.New(fmt.Sprintf("%s-cp", c.Name), c.DataDir)
	token, err := k.GetJoinToken()
	if err != nil {
		return fmt.Errorf("get join token: %w", err)
	}

	// start workers
	for _, node := range c.Nodes {
		if node.Role != RoleWorker {
			continue
		}
		v := vm.New(c.nodeVMConfig(node))
		if err := v.Start(); err != nil {
			return fmt.Errorf("start %s: %w", node.Name, err)
		}
		kw := k0s.New(node.Name, c.DataDir)
		if err := kw.InstallWorker(token); err != nil {
			return fmt.Errorf("install k0s worker: %w", err)
		}
	}

	return nil
}

func (c *Cluster) Stop() error {
	if err := c.load(); err != nil {
		return err
	}
	for _, node := range c.Nodes {
		v := vm.New(c.nodeVMConfig(node))
		if err := v.Stop(); err != nil {
			fmt.Printf("warn: stop %s: %v\n", node.Name, err)
		}
	}
	return nil
}

func (c *Cluster) AddNode() error {
	if err := c.load(); err != nil {
		return err
	}
	idx := len(c.Nodes)
	node := Node{
		Name: fmt.Sprintf("%s-worker-%d", c.Name, idx),
		Role: RoleWorker,
	}
	v := vm.New(c.nodeVMConfig(node))
	if err := v.Create(); err != nil {
		return err
	}
	if err := v.Start(); err != nil {
		return err
	}
	k := k0s.New(fmt.Sprintf("%s-cp", c.Name), c.DataDir)
	token, err := k.GetJoinToken()
	if err != nil {
		return err
	}
	kw := k0s.New(node.Name, c.DataDir)
	if err := kw.InstallWorker(token); err != nil {
		return err
	}
	c.Nodes = append(c.Nodes, node)
	return c.save()
}

func (c *Cluster) RemoveNode(name string) error {
	if err := c.load(); err != nil {
		return err
	}
	for i, node := range c.Nodes {
		if node.Name == name {
			v := vm.New(c.nodeVMConfig(node))
			v.Stop()
			c.Nodes = append(c.Nodes[:i], c.Nodes[i+1:]...)
			return c.save()
		}
	}
	return fmt.Errorf("node %s not found", name)
}

func (c *Cluster) nodeVMConfig(node Node) vm.VMConfig {
	var nodeCfg config.NodeConfig
	if node.Role == RoleControlPlane {
		nodeCfg = c.cfg.Cluster.ControlPlane
	} else {
		nodeCfg = c.cfg.Cluster.Worker
	}
	return vm.VMConfig{
		Name:    node.Name,
		CPU:     nodeCfg.CPU,
		Memory:  nodeCfg.Memory,
		Disk:    nodeCfg.Disk,
		DataDir: filepath.Join(c.DataDir, "nodes"),
	}
}

func (c *Cluster) save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.DataDir, "cluster.json"), data, 0644)
}

func (c *Cluster) load() error {
	data, err := os.ReadFile(filepath.Join(c.DataDir, "cluster.json"))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, c)
}

func List(cfg *config.Config) ([]string, error) {
	dir := filepath.Join(cfg.DataDir, "clusters")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var clusters []string
	for _, e := range entries {
		if e.IsDir() {
			clusters = append(clusters, e.Name())
		}
	}
	return clusters, nil
}
