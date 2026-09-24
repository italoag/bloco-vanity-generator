package tui

import (
	"strings"
	"testing"

	"bloco-vgen/pkg/wallet"
)

func TestEngineInfoIsZero(t *testing.T) {
	if !(EngineInfo{}).IsZero() {
		t.Fatalf("zero EngineInfo should report IsZero=true")
	}
	if (EngineInfo{Engine: "cpu"}).IsZero() {
		t.Fatalf("EngineInfo with engine should report IsZero=false")
	}
	if (EngineInfo{ThreadCount: 2}).IsZero() {
		t.Fatalf("EngineInfo with thread count should report IsZero=false")
	}
}

func TestEngineInfoRowsCPU(t *testing.T) {
	rows := engineInfoRows(EngineInfo{
		Engine:      "cpu",
		ThreadCount: 4,
		Network:     "ethereum",
	})

	labels := make([]string, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, row.label)
	}

	for _, banned := range []string{"Device", "Batch Size", "Validation"} {
		for _, label := range labels {
			if label == banned {
				t.Fatalf("CPU engine info should not include %q row, got labels %v", banned, labels)
			}
		}
	}

	mustContain := []string{"Engine", "Threads", "Network"}
	for _, want := range mustContain {
		found := false
		for _, label := range labels {
			if label == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("CPU engine info missing %q row, got labels %v", want, labels)
		}
	}
}

func TestEngineInfoRowsMetalIncludesDeviceAndBatch(t *testing.T) {
	rows := engineInfoRows(EngineInfo{
		Engine:          "metal",
		RequestedEngine: "auto",
		DeviceName:      "Apple M3 Pro",
		BatchSize:       4096,
		MetalValidation: "full",
		ThreadCount:     8,
		Network:         "ethereum",
	})

	labelValues := make(map[string]string, len(rows))
	for _, row := range rows {
		labelValues[row.label] = row.value
	}

	wantPairs := map[string]string{
		"Engine":     "metal",
		"Requested":  "auto",
		"Device":     "Apple M3 Pro",
		"Batch Size": "4096",
		"Validation": "full",
		"Threads":    "8",
		"Network":    "ethereum",
	}
	for label, want := range wantPairs {
		got, ok := labelValues[label]
		if !ok {
			t.Fatalf("Metal engine info missing %q row, got %v", label, labelValues)
		}
		if got != want {
			t.Fatalf("Metal engine info %q = %q, want %q", label, got, want)
		}
	}
}

func TestEngineInfoRowsSkipsRequestedWhenEqualResolved(t *testing.T) {
	rows := engineInfoRows(EngineInfo{
		Engine:          "cpu",
		RequestedEngine: "cpu",
	})
	for _, row := range rows {
		if row.label == "Requested" {
			t.Fatalf("Requested row should be skipped when equal to Engine, got %v", row)
		}
	}
}

func TestProgressModelRendersEngineBlock(t *testing.T) {
	stats := &wallet.GenerationStats{Pattern: "ab"}
	model := NewProgressModel(stats, nil).WithEngineInfo(EngineInfo{
		Engine:      "metal",
		DeviceName:  "Apple M3 Pro",
		BatchSize:   1024,
		ThreadCount: 4,
	})

	view := model.View()
	for _, want := range []string{"Engine", "metal", "Apple M3 Pro", "1024"} {
		if !strings.Contains(view, want) {
			t.Fatalf("progress view missing %q, view=%s", want, view)
		}
	}
}

func TestProgressModelOmitsEngineBlockWhenZero(t *testing.T) {
	stats := &wallet.GenerationStats{Pattern: "ab"}
	model := NewProgressModel(stats, nil)
	if got := model.renderEngineInfo(); got != "" {
		t.Fatalf("renderEngineInfo() should be empty when EngineInfo is zero, got %q", got)
	}
}

func TestBenchmarkModelRendersEngineBlock(t *testing.T) {
	model := NewBenchmarkModel().WithEngineInfo(EngineInfo{
		Engine:      "metal",
		DeviceName:  "Apple M3 Pro",
		BatchSize:   512,
		ThreadCount: 2,
	})
	block := model.renderEngineInfoBlock("  ")
	for _, want := range []string{"Engine", "metal", "Apple M3 Pro", "512"} {
		if !strings.Contains(block, want) {
			t.Fatalf("benchmark engine block missing %q, block=%s", want, block)
		}
	}
}

func TestBenchmarkModelRendersStateViews(t *testing.T) {
	base := NewBenchmarkModel().WithEngineInfo(EngineInfo{Engine: "cpu", ThreadCount: 2})
	base.results = &wallet.BenchmarkResult{TotalAttempts: 100, ThreadCount: 2}
	base.table.SetRows(base.generateResultsRows())

	cases := []struct {
		state  BenchmarkState
		header string
	}{
		{BenchmarkStateProgress, "Benchmark Running"},
		{BenchmarkStateTransitioning, "Loading Results"},
		{BenchmarkStateResults, "Benchmark Results"},
	}
	for _, tc := range cases {
		model := base
		model.state = tc.state
		view := model.View()
		if !strings.Contains(view, tc.header) {
			t.Fatalf("state %d view missing header %q, view=%s", tc.state, tc.header, view)
		}
		if tc.state == BenchmarkStateResults {
			for _, want := range []string{"Total Attempts", "Engine"} {
				if !strings.Contains(view, want) {
					t.Fatalf("results view missing %q, view=%s", want, view)
				}
			}
		}

		model.quitting = true
		if got := model.View(); got != "" {
			t.Fatalf("state %d quitting view should be empty, got %q", tc.state, got)
		}
	}
}

func TestStatsModelRendersPatternOverview(t *testing.T) {
	model := NewStatsModel(&wallet.GenerationStats{Pattern: "ab", IsChecksum: true, Difficulty: 256, Probability50: 177})
	view := model.View()
	for _, want := range []string{"Pattern Length", "Enabled (increases difficulty)", "Detailed Statistics"} {
		if !strings.Contains(view, want) {
			t.Fatalf("stats view missing %q, view=%s", want, view)
		}
	}
}
