package relay

import (
	"testing"

	"chihqiang/llm-gate/model"
)

func TestCalculateCostCents(t *testing.T) {
	cases := []struct {
		name             string
		prompt, complete int
		ratio, compRatio float64
		basePrice        int64
		want             int64
	}{
		{"base 1:1 ratio, 2分/1k", 1000, 1000, 1.0, 1.0, 2, 4},
		{"prompt only", 10000, 0, 1.0, 1.0, 2, 20},
		{"completion ratio 2x", 1000, 1000, 1.0, 2.0, 2, 6},
		{"model ratio 0.5", 10000, 0, 0.5, 1.0, 2, 10},
		{"round half up", 1500, 0, 1.0, 1.0, 1, 2},
		{"base price default when <=0", 1000, 0, 1.0, 1.0, 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mc := &model.ModelConfig{ModelRatio: c.ratio, CompletionRatio: c.compRatio}
			got := CalculateCostCents(c.prompt, c.complete, mc, c.basePrice)
			if got != c.want {
				t.Fatalf("CalculateCostCents(%d,%d,ratio=%.1f,comp=%.1f,price=%d) = %d, want %d",
					c.prompt, c.complete, c.ratio, c.compRatio, c.basePrice, got, c.want)
			}
		})
	}
}

func TestRequestIDsOf(t *testing.T) {
	batch := []model.UsageLog{
		{RequestID: "r1"},
		{RequestID: "r1"},
		{RequestID: ""},
		{RequestID: "r2"},
		{RequestID: "r3"},
		{RequestID: "r4"},
		{RequestID: "r5"},
		{RequestID: "r6"},
		{RequestID: "r7"},
		{RequestID: "r8"},
		{RequestID: "r9"},
	}
	got := requestIDsOf(batch)
	// 去重：r1 只出现一次；截断到 8 个；空 request_id 跳过
	if got == "r1,r1" {
		t.Fatalf("requestIDsOf should deduplicate: %q", got)
	}
	want := "r1,r2,r3,r4,r5,r6,r7,r8"
	if got != want {
		t.Fatalf("requestIDsOf = %q, want %q", got, want)
	}
}
