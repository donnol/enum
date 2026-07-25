package enum_test

import (
	"testing"

	"github.com/donnol/enum"
)

// ── type definitions for benchmarks ──────────────────────────────────

type BenchInt int
type BenchIntEnum struct {
	enum.Struct[BenchInt]
	A BenchInt `enum:"0,A"`
	B BenchInt `enum:"1,B"`
	C BenchInt `enum:"2,C"`
	D BenchInt `enum:"3,D"`
	E BenchInt `enum:"4,E"`
}

type BenchStr string
type BenchStrEnum struct {
	enum.Struct[BenchStr]
	Red    BenchStr `enum:",红色"`
	Green  BenchStr `enum:",绿色"`
	Blue   BenchStr `enum:",蓝色"`
	Yellow BenchStr `enum:",黄色"`
	Purple BenchStr `enum:",紫色"`
}

type BenchPlainInt struct {
	enum.Struct[int]
	A int `enum:"0,A"`
	B int `enum:"1,B"`
	C int `enum:"2,C"`
	D int `enum:"3,D"`
	E int `enum:"4,E"`
}

type BenchPlainStr struct {
	enum.Struct[string]
	Red    string `enum:",红色"`
	Green  string `enum:",绿色"`
	Blue   string `enum:",蓝色"`
	Yellow string `enum:",黄色"`
	Purple string `enum:",紫色"`
}

// ── InitStruct ───────────────────────────────────────────────────────

func BenchmarkStruct_Init_CustomInt(b *testing.B) {
	for b.Loop() {
		_ = enum.InitFor[BenchInt, BenchIntEnum]()
	}
}

func BenchmarkStruct_Init_CustomStr(b *testing.B) {
	for b.Loop() {
		_ = enum.InitFor[BenchStr, BenchStrEnum]()
	}
}

func BenchmarkStruct_Init_PlainInt(b *testing.B) {
	for b.Loop() {
		_ = enum.InitFor[int, BenchPlainInt]()
	}
}

func BenchmarkStruct_Init_PlainStr(b *testing.B) {
	for b.Loop() {
		_ = enum.InitFor[string, BenchPlainStr]()
	}
}

// ── ByKey lookup ────────────────────────────────────────────────────

func BenchmarkStruct_ByKey_CustomInt(b *testing.B) {
	var e = enum.InitFor[BenchInt, BenchIntEnum]()
	b.ResetTimer()
	for b.Loop() {
		e.ByKey("C")
	}
}

func BenchmarkStruct_ByKey_CustomStr(b *testing.B) {
	var e = enum.InitFor[BenchStr, BenchStrEnum]()
	b.ResetTimer()
	for b.Loop() {
		e.ByKey("Blue")
	}
}

func BenchmarkStruct_ByKey_PlainInt(b *testing.B) {
	var e = enum.InitFor[int, BenchPlainInt]()
	b.ResetTimer()
	for b.Loop() {
		e.ByKey("C")
	}
}

func BenchmarkStruct_ByKey_PlainStr(b *testing.B) {
	var e = enum.InitFor[string, BenchPlainStr]()
	b.ResetTimer()
	for b.Loop() {
		e.ByKey("Blue")
	}
}

// ── ByValue lookup ───────────────────────────────────────────────────

func BenchmarkStruct_ByValue_CustomInt(b *testing.B) {
	var e = enum.InitFor[BenchInt, BenchIntEnum]()
	b.ResetTimer()
	for b.Loop() {
		e.ByValue(BenchInt(2))
	}
}

func BenchmarkStruct_ByValue_CustomStr(b *testing.B) {
	var e = enum.InitFor[BenchStr, BenchStrEnum]()
	b.ResetTimer()
	for b.Loop() {
		e.ByValue(BenchStr("BLUE"))
	}
}

func BenchmarkStruct_ByValue_PlainInt(b *testing.B) {
	var e = enum.InitFor[int, BenchPlainInt]()
	b.ResetTimer()
	for b.Loop() {
		e.ByValue(2)
	}
}

func BenchmarkStruct_ByValue_PlainStr(b *testing.B) {
	var e = enum.InitFor[string, BenchPlainStr]()
	b.ResetTimer()
	for b.Loop() {
		e.ByValue("BLUE")
	}
}

// ── Range iteration ──────────────────────────────────────────────────

func BenchmarkStruct_Range_CustomInt(b *testing.B) {
	var e = enum.InitFor[BenchInt, BenchIntEnum]()
	b.ResetTimer()
	for b.Loop() {
		for _, item := range e.Range() {
			_ = item.Name()
			_ = item.Value()
		}
	}
}

func BenchmarkStruct_Range_CustomStr(b *testing.B) {
	var e = enum.InitFor[BenchStr, BenchStrEnum]()
	b.ResetTimer()
	for b.Loop() {
		for _, item := range e.Range() {
			_ = item.Name()
			_ = item.Value()
		}
	}
}

func BenchmarkStruct_Range_PlainInt(b *testing.B) {
	var e = enum.InitFor[int, BenchPlainInt]()
	b.ResetTimer()
	for b.Loop() {
		for _, item := range e.Range() {
			_ = item.Name()
			_ = item.Value()
		}
	}
}

func BenchmarkStruct_Range_PlainStr(b *testing.B) {
	var e = enum.InitFor[string, BenchPlainStr]()
	b.ResetTimer()
	for b.Loop() {
		for _, item := range e.Range() {
			_ = item.Name()
			_ = item.Value()
		}
	}
}

// ── Contains ─────────────────────────────────────────────────────────

func BenchmarkStruct_Contains_CustomInt(b *testing.B) {
	var e = enum.InitFor[BenchInt, BenchIntEnum]()
	b.ResetTimer()
	for b.Loop() {
		e.Contains(BenchInt(2))
	}
}

func BenchmarkStruct_Contains_CustomStr(b *testing.B) {
	var e = enum.InitFor[BenchStr, BenchStrEnum]()
	b.ResetTimer()
	for b.Loop() {
		e.Contains(BenchStr("BLUE"))
	}
}

func BenchmarkStruct_Contains_PlainInt(b *testing.B) {
	var e = enum.InitFor[int, BenchPlainInt]()
	b.ResetTimer()
	for b.Loop() {
		e.Contains(2)
	}
}

func BenchmarkStruct_Contains_PlainStr(b *testing.B) {
	var e = enum.InitFor[string, BenchPlainStr]()
	b.ResetTimer()
	for b.Loop() {
		e.Contains("BLUE")
	}
}
