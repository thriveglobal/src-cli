package spy

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/olekukonko/tablewriter"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/src-cli/internal/scout"
)

// MockResourceAverages represents a mock implementation of the ResourceAverages struct.
type MockResourceAverages struct {
	PodName         string
	CpuAverageUsage float64
	MemAverageUsage float64
}

// MockChannel is a mock implementation of the channel that receives ResourceAverages.
type MockChannel struct {
	Data     []MockResourceAverages
	Position int
}

func TestGetPodUsage(t *testing.T) {
	cases := []struct {
		name         string
		usageMetrics []scout.UsageStats
		want         float64
		kind         string
	}{
		{
			name: "return correct cpu usage for pod with multiple containers",
			usageMetrics: []scout.UsageStats{
				{
					ContainerName: "container1",
					CpuUsage:      2.5,
					MemoryUsage:   22.3,
				},
				{
					ContainerName: "container2",
					CpuUsage:      3.5,
					MemoryUsage:   17.32,
				},
			},
			want: 6.0,
			kind: scout.CPU,
		},
		{
			name: "return correct cpu usage for pod with single container",
			usageMetrics: []scout.UsageStats{
				{
					ContainerName: "container1",
					CpuUsage:      2.5,
					MemoryUsage:   45.0,
				},
			},
			want: 2.5,
			kind: scout.CPU,
		},
		{
			name: "return correct memory usage for pod with single container",
			usageMetrics: []scout.UsageStats{
				{
					ContainerName: "container1",
					CpuUsage:      2.5,
					MemoryUsage:   45.0,
				},
			},
			want: 45.0,
			kind: scout.MEMORY,
		},
		{
			name: "return correct memory usage for pod with multi container",
			usageMetrics: []scout.UsageStats{
				{
					ContainerName: "container1",
					CpuUsage:      13.0,
					MemoryUsage:   2.63,
				},
				{
					ContainerName: "container2",
					CpuUsage:      15.5,
					MemoryUsage:   45.0,
				},
				{
					ContainerName: "container3",
					CpuUsage:      64.22,
					MemoryUsage:   31.06,
				},
			},
			want: 78.69,
			kind: scout.MEMORY,
		},
	}

	for _, tc := range cases {
		tc := tc
		got := mock_getPodUsage(tc.usageMetrics, tc.kind)
		if got != tc.want {
			t.Errorf("got %.2f, want %.2f", got, tc.want)
		}
	}
}

func TestOutputTableToFile(t *testing.T) {
	data := []ResourceAverages{
		{
			PodName:         "Pod 1",
			CpuAverageUsage: 50.0,
			MemAverageUsage: 30.0,
		},
		{
			PodName:         "Pod 2",
			CpuAverageUsage: 60.5,
			MemAverageUsage: 40.2,
		},
	}

	ch := make(chan ResourceAverages, 2)
	for _, d := range data {
		go func(d ResourceAverages) {
			fmt.Println(d)
			ch <- d
		}(d)
	}

	go func(ch chan ResourceAverages) {
		err := outputTableToFile(ch, 2)
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
	}(ch)

	_, err := os.Stat("/tmp/resource-averages.txt")
	if os.IsNotExist(err) {
		t.Error("Expected file to be created, but it does not exist")
	}

	fileContent, err := ioutil.ReadFile("/tmp/resource-averages.txt")
	fmt.Println(string(fileContent))
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}

	expectedContent := `+-------+----------+----------+
|  POD  | CPU AVG% | MEM AVG% |
+-------+----------+----------+
| Pod 1 | 50.00%   | 30.00%   |
| Pod 2 | 60.50%   | 40.20%   |
+-------+----------+----------+
`
	if strings.TrimSpace(string(fileContent)) != strings.TrimSpace(expectedContent) {
		t.Errorf("File content doesn't match the expected content:\n\nExpected:\n%s\n\nActual:\n%s",
			expectedContent, string(fileContent))
	}
}

func mock_getPodUsage(containers []scout.UsageStats, kind string) (usage float64) {
	for _, container := range containers {
		if kind == scout.CPU {
			usage += container.CpuUsage
		} else if kind == scout.MEMORY {
			usage += container.MemoryUsage
		} else {
			fmt.Printf("%s is an invalid argument for 'kind', use '%s' or '%s'", kind, scout.MEMORY, scout.CPU)
			os.Exit(1)
		}
	}
	return
}
