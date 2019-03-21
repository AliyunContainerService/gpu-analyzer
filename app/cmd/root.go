package cmd

import (
	"context"

	"github.com/AliyunContainerService/gpu-analyzer/pkg/prom"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:  "gpu-analyzer",
	Long: `GPU analyzer for k8s`,
	Run:  rootRun,
}

func init() {
	rootCmd.PersistentFlags().String("prom-url", "http://127.0.0.1:9090", "The URL of Prometheus HTTP API endpoint")
	rootCmd.PersistentFlags().String("report-path", "_analyzer_report/gpu-analysis.csv", "")
	viper.BindPFlag("prom-url", rootCmd.PersistentFlags().Lookup("prom-url"))
	viper.BindPFlag("report-path", rootCmd.PersistentFlags().Lookup("report-path"))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func rootRun(_ *cobra.Command, _ []string) {
	logrus.Infof("prom url: %q", viper.Get("prom-url").(string))

	qw, err := prom.NewQueryWorker()
	if err != nil {
		logrus.Fatal(err)
	}
	err = qw.GetPodsGPUMemUtil(context.TODO())
	if err != nil {
		logrus.Fatal(err)
	}
}
