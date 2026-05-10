package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type NodeConfig struct {
	CPU    int    `yaml:"cpu"`
	Memory string `yaml:"memory"`
	Disk   string `yaml:"disk"`
}

type ClusterConfig struct {
	ControlPlane NodeConfig `yaml:"control_plane"`
	Worker       NodeConfig `yaml:"worker"`
	WorkerCount  int        `yaml:"worker_count"`
}

type Config struct {
	MaxClusters int           `yaml:"max_clusters"`
	Cluster     ClusterConfig `yaml:"cluster"`
	DataDir     string        `yaml:"data_dir"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		MaxClusters: 0,
		DataDir:     filepath.Join(home, ".kube-server"),
		Cluster: ClusterConfig{
			WorkerCount: 2,
			ControlPlane: NodeConfig{
				CPU:    2,
				Memory: "2GB",
				Disk:   "20GB",
			},
			Worker: NodeConfig{
				CPU:    2,
				Memory: "2GB",
				Disk:   "10GB",
			},
		},
	}
}

func Load() (*Config, error) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".kube-server", "config.yaml")

	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	return cfg, yaml.Unmarshal(data, cfg)
}

func (c *Config) Save() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".kube-server")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0644)
}
