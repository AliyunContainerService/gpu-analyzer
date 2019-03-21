package prom

type PodRecord struct {
	Name           string
	Namespace      string
	PeakGPUMemory  float64
	TotalGPUMemory float64
	gmemUtilization float64
}
