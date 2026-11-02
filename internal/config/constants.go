package config

const (
	unitsBytes             = "B"
	unitsKilobytes         = "KiB"
	unitsMegabytes         = "MiB"
	unitsGigabytes         = "GiB"
	cpuPeriod       uint64 = 100_000
	unitsMillicores        = 'm'
)

var unitsArr = []string{unitsBytes, unitsKilobytes, unitsMegabytes, unitsGigabytes}
