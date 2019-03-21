package prom

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"sort"
	"time"

	"github.com/spf13/viper"

	"github.com/sirupsen/logrus"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type QueryWorker struct {
	pcli promapi.Client
}

func NewQueryWorker() (*QueryWorker, error) {
	cfg := promapi.Config{
		Address: viper.Get("prom-url").(string),
	}
	pcli, err := promapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &QueryWorker{
		pcli: pcli,
	}, nil
}

// Get pods' gpu memory utilization in sorted order from low to high.
// 1. range query total memory and iterate results to get each pod's total gpu memory
// 2. range query used memory and iterate results to get each pod's peak gpu memory usage
func (w *QueryWorker) GetPodsGPUMemUtil(ctx context.Context) error {
	papi := promv1.NewAPI(w.pcli)

	total, err := papi.QueryRange(ctx, "nvidia_gpu_memory_total_bytes", promV1RangeUsedMemory())
	if err != nil {
		return err
	}
	totalByPod := getPodsTotalGPUMem(total.(model.Matrix))

	used, err := papi.QueryRange(ctx, "nvidia_gpu_memory_used_bytes", promV1RangeUsedMemory())
	if err != nil {
		return err
	}

	return writeRecords(getPodsGMemUtil(used.(model.Matrix), totalByPod), viper.Get("report-path").(string))
}

func writeRecords(rs []*PodRecord, reportPath string) error {
	logrus.Infof("writing records to file (%s)", reportPath)
	f, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	w.WriteString("Pod,util,real,request\n")
	for _, r := range rs {
		s := fmt.Sprintf("%s,%.3f,%d,%d\n", path.Join(r.Namespace, r.Name), r.gmemUtilization, int64(r.PeakGPUMemory), int64(r.TotalGPUMemory))
		w.WriteString(s)
	}
	return nil
}

func promV1RangeUsedMemory() promv1.Range {
	now := time.Now()
	start := now.Add(-12 * time.Hour)
	step := 3 * time.Minute // 240 records per pod
	return promv1.Range{
		Start: start,
		End:   now,
		Step:  step,
	}
}

func getPodsTotalGPUMem(m model.Matrix) map[string]float64 {
	totalGMemByPod := make(map[string]float64)
	for _, sampleStream := range m {
		podname := string(sampleStream.Metric["pod_name"])
		if len(podname) == 0 {
			continue
		}

		totalGMem := float64(0)
		for _, samplePair := range sampleStream.Values {
			v := float64(samplePair.Value)
			if v > totalGMem {
				totalGMem = v
			}
		}
		namespace := string(sampleStream.Metric["namespace_name"])
		totalGMemByPod[path.Join(namespace, podname)] = totalGMem
	}
	return totalGMemByPod
}

func getPodsGMemUtil(m model.Matrix, totalByPod map[string]float64) []*PodRecord {
	prByPod := make(map[string]*PodRecord, len(m))
	for _, sampleStream := range m {
		name := string(sampleStream.Metric["pod_name"])
		if len(name) == 0 {
			continue
		}
		namespace := string(sampleStream.Metric["namespace_name"])
		uniqueName := path.Join(namespace, name)
		total, ok := totalByPod[uniqueName]
		if !ok {
			logrus.Infof("Pod (%s) not found in total_memory", uniqueName)
			continue
		}

		peak := float64(0)
		for _, samplePair := range sampleStream.Values {
			v := float64(samplePair.Value)
			if v > peak {
				peak = v
			}
		}

		r := &PodRecord{
			Name:            name,
			Namespace:       namespace,
			PeakGPUMemory:   peak,
			TotalGPUMemory:  total,
			gmemUtilization: peak / total,
		}

		exist, ok := prByPod[uniqueName]
		if ok && exist.PeakGPUMemory > r.PeakGPUMemory {
			continue
		}
		prByPod[uniqueName] = r
	}
	prs := make([]*PodRecord, 0, len(prByPod))
	for _, r := range prByPod {
		prs = append(prs, r)
	}
	sort.Slice(prs, func(i, j int) bool {
		return prs[i].gmemUtilization < prs[j].gmemUtilization
	})
	return prs
}
