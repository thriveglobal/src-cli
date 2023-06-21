package spy

import (
	"fmt"
	"os"
	"testing"

	"github.com/sourcegraph/src-cli/internal/scout"
)

func TestGetPodCPUUsage(t *testing.T) {
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
