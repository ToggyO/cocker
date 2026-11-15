package config

import (
	"path/filepath"
	"slices"

	configErrors "github.com/ToggyO/cocker/internal/config/errors"
	"github.com/ToggyO/cocker/internal/domain"
	"github.com/ToggyO/cocker/internal/ptr"
)

const slash = "/"

// Normalize applies defaults and converts input into an independent domain spec.
// It returns the first conversion error. The caller must validate the result.
func Normalize(config Input) (domain.ContainerSpec, error) {
	nzer := normalizer{
		config: &config,
		spec: domain.ContainerSpec{
			Namespaces: domain.DefaultNamespaceSpec(),
			IO:         domain.DefaultIOSpec(),
		},
	}
	return nzer.Normalize()
}

type normalizer struct {
	config *Input
	spec   domain.ContainerSpec
}

func (n *normalizer) Normalize() (domain.ContainerSpec, error) {
	var err error

	if err = n.normalizeRootFS(); err != nil {
		return n.spec, err
	}

	n.setCommand()
	n.normalizeHostname()
	n.normalizeWorkingDir()
	n.setEnvs()

	if err = n.normalizeResources(); err != nil {
		return n.spec, err
	}

	return n.spec, nil
}

func (n *normalizer) normalizeRootFS() error {
	if n.config.RootFS == nil || *n.config.RootFS == "" {
		n.spec.RootFS = ""
		return nil
	}

	path, err := filepath.Abs(*n.config.RootFS)
	if err != nil {
		return configErrors.NewFieldError("rootfs", err)
	}

	n.spec.RootFS = path
	return nil
}

func (n *normalizer) setCommand() {
	if n.config.Command != nil {
		n.spec.Command = slices.Clone(n.config.Command)
	}
}

func (n *normalizer) normalizeHostname() {
	if n.config.Hostname == nil {
		n.spec.Hostname = ""
		return
	}
	n.spec.Hostname = *n.config.Hostname
}

func (n *normalizer) normalizeWorkingDir() {
	if n.config.WorkingDir == nil || *n.config.WorkingDir == "" {
		n.spec.WorkingDir = slash
		return
	}
	cleanFilePath := filepath.Clean(*n.config.WorkingDir)
	n.spec.WorkingDir = cleanFilePath
}

func (n *normalizer) setEnvs() {
	for _, env := range n.config.Env {
		n.spec.Env = append(n.spec.Env, domain.EnvVar{Name: env.Name, Value: env.Value})
	}
}

func (n *normalizer) normalizeResources() error {
	if n.config.Resources == nil {
		return nil
	}

	if n.config.Resources.Memory != nil {
		val, err := ParseMemoryBytes(*n.config.Resources.Memory)
		if err != nil {
			return configErrors.NewFieldError("resources.memory", err)
		}
		n.spec.Resources.MemoryBytes = ptr.To(val)
	}

	if n.config.Resources.CPUs != nil {
		val, err := ParseCPUQuota(*n.config.Resources.CPUs)
		if err != nil {
			return configErrors.NewFieldError("resources.cpus", err)
		}
		n.spec.Resources.CPUQuota = ptr.To(val)
	}

	if n.config.Resources.Pids != nil {
		n.spec.Resources.PidsLimit = ptr.To(*n.config.Resources.Pids)
	}

	return nil
}
