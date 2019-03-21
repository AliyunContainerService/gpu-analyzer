package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var reportPath string

func init() {
	flag.StringVar(&reportPath, "report-path", "_analyzer_report/gpu-analysis.csv", "")
	flag.Parse()
}

func main() { // check cache
	// cache not exist, fetch cache
	// print
	f, err := os.Open(reportPath)
	if err != nil {
		logrus.Fatal(err)
	}

	r := csv.NewReader(bufio.NewReader(f))

	name := flag.Arg(0)
	fmt.Printf("Pod,request,real\n")

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Fatal(err)
		}
		if record[0] == name {
			fmt.Printf("%s,%s,%s\n", name, record[3], record[2])
		}
	}
}
