package config

// Input holds configuration values before merging and normalization.
// Nil fields represent unspecified values; JSON null is treated as absent.
type Input struct {
	RootFS     *string         `json:"rootfs"`
	Command    []string        `json:"command"`
	Hostname   *string         `json:"hostname"`
	WorkingDir *string         `json:"workingDir"`
	Env        []envVar        `json:"env"`
	Resources  *resourcesInput `json:"resources"`
}

type envVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type resourcesInput struct {
	Memory *string `json:"memory"`
	CPUs   *string `json:"cpus"`
	Pids   *uint64 `json:"pids"`
}
