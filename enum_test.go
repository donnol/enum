package enum_test

import (
	"slices"
	"testing"
	"time"

	"github.com/donnol/enum"
)

// ── type 1: library int-based type ──────────────────────────────────

var WeekdayEnum = enum.New(
	enum.ItemFrom("Sunday", "周日", time.Sunday),
	enum.ItemFrom("Monday", "周一", time.Monday),
	enum.ItemFrom("Tuesday", "周二", time.Tuesday),
	enum.ItemFrom("Wednesday", "周三", time.Wednesday),
	enum.ItemFrom("Thursday", "周四", time.Thursday),
	enum.ItemFrom("Friday", "周五", time.Friday),
	enum.ItemFrom("Saturday", "周六", time.Saturday),
)

// ── type 2: custom int type + iota constants ─────────────────────────
// This is the pattern that gives you native switch-case behaviour:
// define typed constants with iota, then register them in an enum only
// when you need names, iteration, or validation.

type Status int

const (
	StatusPending   Status = iota // 0
	StatusActive                  // 1
	StatusSuspended               // 2
	StatusDeleted                 // 3
)

var StatusEnum = enum.New(
	enum.ItemFrom("Pending", "待处理", StatusPending),
	enum.ItemFrom("Active", "活跃", StatusActive),
	enum.ItemFrom("Suspended", "已暂停", StatusSuspended),
	enum.ItemFrom("Deleted", "已删除", StatusDeleted),
)

// ── type 3: custom string type ───────────────────────────────────────

type Color string

const (
	ColorRed    Color = "RED"
	ColorGreen  Color = "GREEN"
	ColorBlue   Color = "BLUE"
	ColorYellow Color = "YELLOW"
)

var ColorEnum = enum.New(
	enum.ItemFrom("Red", "红色", ColorRed),
	enum.ItemFrom("Green", "绿色", ColorGreen),
	enum.ItemFrom("Blue", "蓝色", ColorBlue),
	enum.ItemFrom("Yellow", "黄色", ColorYellow),
)

// ── type 4: plain int ────────────────────────────────────────────────

var SizeEnum = enum.New(
	enum.ItemFrom("Small", "小", 1),
	enum.ItemFrom("Medium", "中", 2),
	enum.ItemFrom("Large", "大", 3),
)

// =====================================================================
// Tests
// =====================================================================

func TestEnum_Len(t *testing.T) {
	if WeekdayEnum.Len() != 7 {
		t.Errorf("Len = %d, want 7", WeekdayEnum.Len())
	}
	if StatusEnum.Len() != 4 {
		t.Errorf("Len = %d, want 4", StatusEnum.Len())
	}
}

func TestEnum_ByKey_Found(t *testing.T) {
	item, ok := WeekdayEnum.ByKey("Tuesday")
	if !ok {
		t.Fatal("ByKey(Tuesday) not found")
	}
	if item.Key() != "Tuesday" {
		t.Errorf("Name = %q, want Tuesday", item.Key())
	}
	if item.Name() != "周二" {
		t.Errorf("Name = %q, want 周二", item.Name())
	}
	if item.Value() != time.Tuesday {
		t.Errorf("Value = %v, want %v", item.Value(), time.Tuesday)
	}
}

func TestEnum_ByKey_StringType(t *testing.T) {
	item, ok := ColorEnum.ByKey("Blue")
	if !ok {
		t.Fatal("ByKey(Blue) not found")
	}
	if item.Name() != "蓝色" {
		t.Errorf("Name = %q, want 蓝色", item.Name())
	}
	if item.Value() != ColorBlue {
		t.Errorf("Value = %v, want %v", item.Value(), ColorBlue)
	}
}

func TestEnum_ByKey_CustomIntType(t *testing.T) {
	item, ok := StatusEnum.ByKey("Suspended")
	if !ok {
		t.Fatal("ByKey(Suspended) not found")
	}
	if item.Name() != "已暂停" {
		t.Errorf("Name = %q, want 已暂停", item.Name())
	}
	if item.Value() != StatusSuspended {
		t.Errorf("Value = %v, want %v", item.Value(), StatusSuspended)
	}
}

func TestEnum_ByKey_PlainIntType(t *testing.T) {
	item, ok := SizeEnum.ByKey("Large")
	if !ok {
		t.Fatal("ByKey(Large) not found")
	}
	if item.Value() != 3 {
		t.Errorf("Value = %v, want 3", item.Value())
	}
}

func TestEnum_ByKey_NotFound(t *testing.T) {
	_, ok := WeekdayEnum.ByKey("Notaday")
	if ok {
		t.Error("ByKey(Notaday) should not be found")
	}
}

func TestEnum_ByValue_Found(t *testing.T) {
	item, ok := WeekdayEnum.ByValue(time.Friday)
	if !ok {
		t.Fatal("ByValue(Friday) not found")
	}
	if item.Value() != time.Friday {
		t.Errorf("Value = %v, want %v", item.Value(), time.Friday)
	}
}

func TestEnum_ByValue_NotFound(t *testing.T) {
	_, ok := WeekdayEnum.ByValue(time.Weekday(99))
	if ok {
		t.Error("ByValue(99) should not be found")
	}
}

func TestEnum_Contains(t *testing.T) {
	if !WeekdayEnum.Contains(time.Monday) {
		t.Error("Contains(Monday) = false, want true")
	}
	if WeekdayEnum.Contains(time.Weekday(99)) {
		t.Error("Contains(99) = true, want false")
	}
}

func TestEnum_Contains_CustomInt(t *testing.T) {
	if !StatusEnum.Contains(StatusActive) {
		t.Error("Contains(Active) = false, want true")
	}
	if StatusEnum.Contains(Status(99)) {
		t.Error("Contains(99) = true, want false")
	}
	// Zero value of the type (0 = StatusPending) IS a member.
	if !StatusEnum.Contains(Status(0)) {
		t.Error("Contains(0) = false, want true (StatusPending == 0)")
	}
	// After last iota constant.
	if StatusEnum.Contains(StatusDeleted + 1) {
		t.Error("Contains after last const = true, want false")
	}
	// Negative int for a custom int type.
	if StatusEnum.Contains(Status(-1)) {
		t.Error("Contains(-1) = true, want false")
	}
}

func TestEnum_Contains_StringType(t *testing.T) {
	// Zero value "" is not in this enum.
	if ColorEnum.Contains(Color("")) {
		t.Error(`Contains("") = true, want false`)
	}
	// Valid.
	if !ColorEnum.Contains(ColorRed) {
		t.Error("Contains(Red) = false, want true")
	}
	// Same underlying string but not a declared constant.
	if ColorEnum.Contains(Color("BLACK")) {
		t.Error("Contains(BLACK) = true, want false")
	}
}

func TestIn(t *testing.T) {
	// Instance-method-style spelling.
	if !StatusEnum.Contains(StatusPending) {
		t.Error("In(Pending) = false, want true")
	}
	if StatusEnum.Contains(Status(99)) {
		t.Error("In(99) = true, want false")
	}
	if !ColorEnum.Contains(ColorBlue) {
		t.Error("In(ColorBlue) = false, want true")
	}
	if ColorEnum.Contains(Color("BLACK")) {
		t.Error("In(BLACK) = true, want false")
	}
	// Plain int type.
	if !SizeEnum.Contains(2) {
		t.Error("In(2, SizeEnum) = false, want true")
	}
	if SizeEnum.Contains(99) {
		t.Error("In(99, SizeEnum) = true, want false")
	}
}

func TestField(t *testing.T) {
	item, ok := StatusEnum.ByValue(StatusPending)
	if !ok || item.Name() != "待处理" {
		t.Errorf("Field(Pending) = %q, %v, want 待处理, true", item.Name(), ok)
	}
	colorItem, ok := ColorEnum.ByValue(ColorGreen)
	if !ok || colorItem.Name() != "绿色" {
		t.Errorf("Field(Green) = %q, %v, want 绿色, true", colorItem.Name(), ok)
	}
	// Unknown value.
	_, ok = StatusEnum.ByValue(Status(99))
	if ok {
		t.Error("Field(99) should not be found")
	}
}

func TestMustField_Panics(t *testing.T) {
	if item := StatusEnum.MustByValue(StatusActive); item.Key() != "Active" {
		t.Errorf("MustField(Active) = %q, want Active", item.Key())
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustField(99) should panic")
		}
	}()
	StatusEnum.MustByValue(Status(99))
}

func TestEnum_MustByKey(t *testing.T) {
	item := WeekdayEnum.MustByKey("Sunday")
	if item.Value() != time.Sunday {
		t.Errorf("Value = %v, want Sunday", item.Value())
	}
}

func TestEnum_MustByKey_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustByKey(notaday) should panic")
		}
	}()
	WeekdayEnum.MustByKey("notaday")
}

func TestEnum_MustByValue(t *testing.T) {
	item := WeekdayEnum.MustByValue(time.Saturday)
	if item.Key() != "Saturday" {
		t.Errorf("Name = %q, want Saturday", item.Key())
	}
}

func TestEnum_MustByValue_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustByValue(99) should panic")
		}
	}()
	WeekdayEnum.MustByValue(time.Weekday(99))
}

func TestEnum_Index_Found(t *testing.T) {
	first, ok := WeekdayEnum.Index(0)
	if !ok || first.Key() != "Sunday" {
		t.Errorf("Index(0) = %q, %v, want Sunday, true", first.Key(), ok)
	}
	last, ok := WeekdayEnum.Index(WeekdayEnum.Len() - 1)
	if !ok || last.Key() != "Saturday" {
		t.Errorf("Index(Len()-1) = %q, %v, want Saturday, true", last.Key(), ok)
	}
	mid, ok := WeekdayEnum.Index(3)
	if !ok || mid.Key() != "Wednesday" {
		t.Errorf("Index(3) = %q, %v, want Wednesday, true", mid.Key(), ok)
	}
}

func TestEnum_Index_OutOfRange(t *testing.T) {
	if _, ok := WeekdayEnum.Index(-1); ok {
		t.Error("Index(-1) should not be found")
	}
	if _, ok := WeekdayEnum.Index(-999); ok {
		t.Error("Index(-999) should not be found")
	}
	if _, ok := WeekdayEnum.Index(WeekdayEnum.Len()); ok {
		t.Error("Index(Len()) should not be found")
	}
	if _, ok := WeekdayEnum.Index(WeekdayEnum.Len() + 1); ok {
		t.Error("Index(Len()+1) should not be found")
	}
	if _, ok := WeekdayEnum.Index(9999); ok {
		t.Error("Index(9999) should not be found")
	}
}

func TestEnum_MustIndex_Bounds(t *testing.T) {
	// Hit every index from 0 to Len()-1 — must not panic.
	for i := range WeekdayEnum.Len() {
		item := WeekdayEnum.MustIndex(i)
		if item.Key() == "" {
			t.Errorf("MustIndex(%d) returned empty name", i)
		}
	}
}

func TestEnum_MustIndex_Panics(t *testing.T) {
	tests := []struct {
		label string
		i     int
	}{
		{"Len()", WeekdayEnum.Len()},
		{"negative", -1},
		{"far negative", -999},
		{"far positive", 9999},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("MustIndex(%d) should panic", tt.i)
				}
			}()
			WeekdayEnum.MustIndex(tt.i)
		})
	}
}

func TestEnum_All(t *testing.T) {
	all := WeekdayEnum.All()
	if len(all) != 7 {
		t.Errorf("len = %d, want 7", len(all))
	}
	if all[0].Key() != "Sunday" || all[6].Key() != "Saturday" {
		t.Error("order does not match definition order")
	}
	// Verify All returns a copy, not the internal slice.
	originalFirst := WeekdayEnum.All()[0].Key()
	all[0] = enum.ItemFrom("Fakeday", "假", time.Weekday(99))
	if WeekdayEnum.All()[0].Key() != originalFirst {
		t.Error("All() returned the internal slice — enum is mutable from outside")
	}
}

func TestEnum_Names(t *testing.T) {
	names := WeekdayEnum.Keys()
	want := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if !slices.Equal(names, want) {
		t.Errorf("Names = %v, want %v", names, want)
	}
}

func TestEnum_Values(t *testing.T) {
	values := WeekdayEnum.Values()
	if len(values) != 7 {
		t.Fatalf("len = %d, want 7", len(values))
	}
	if values[0] != time.Sunday || values[6] != time.Saturday {
		t.Error("order does not match definition order")
	}
}

func TestEnum_Range(t *testing.T) {
	count := 0
	for name, item := range WeekdayEnum.Range() {
		if name != item.Key() {
			t.Errorf("name %q != item.Key() %q", name, item.Key())
		}
		if !WeekdayEnum.Contains(item.Value()) {
			t.Errorf("value %v from Range is not in the enum", item.Value())
		}
		count++
	}
	if count != 7 {
		t.Errorf("Range iterated %d times, want 7", count)
	}
}

func TestEnum_Range_EarlyBreak(t *testing.T) {
	count := 0
	for _, item := range WeekdayEnum.Range() {
		count++
		if item.Value() == time.Wednesday {
			break
		}
	}
	if count != 4 {
		t.Errorf("Range early-break iterated %d, want 4", count)
	}
}

// ── Switch simulation — typed constants + native switch ──────────────
// Define typed iota/string constants, then switch on them as you would
// with any Go constants. The enum provides metadata, iteration, and
// validation on top — but the switch itself is plain Go.

func statusLabel(s Status) string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusActive:
		return "active"
	case StatusSuspended:
		return "suspended"
	case StatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

func colorHex(c Color) string {
	switch c {
	case ColorRed:
		return "#FF0000"
	case ColorGreen:
		return "#00FF00"
	case ColorBlue:
		return "#0000FF"
	case ColorYellow:
		return "#FFFF00"
	default:
		return "#000000"
	}
}

func TestSwitch_Status(t *testing.T) {
	tests := []struct {
		val  Status
		want string
	}{
		{StatusPending, "pending"},
		{StatusActive, "active"},
		{StatusSuspended, "suspended"},
		{StatusDeleted, "deleted"},
		{Status(99), "unknown"},
	}
	for _, tt := range tests {
		got := statusLabel(tt.val)
		if got != tt.want {
			t.Errorf("statusLabel(%v) = %q, want %q", tt.val, got, tt.want)
		}
	}
}

func TestSwitch_Color(t *testing.T) {
	if got := colorHex(ColorGreen); got != "#00FF00" {
		t.Errorf("colorHex(Green) = %q, want #00FF00", got)
	}
	if got := colorHex(Color("PURPLE")); got != "#000000" {
		t.Errorf("colorHex(PURPLE) = %q, want #000000", got)
	}
}

// Verify the constants match the enum values (they must be kept in sync).
func TestSwitch_ConstantsMatchEnum(t *testing.T) {
	// Every constant must exist in the enum.
	for _, want := range []Status{StatusPending, StatusActive, StatusSuspended, StatusDeleted} {
		if !StatusEnum.Contains(want) {
			t.Errorf("StatusEnum missing constant %v", want)
		}
	}
	for _, want := range []Color{ColorRed, ColorGreen, ColorBlue, ColorYellow} {
		if !ColorEnum.Contains(want) {
			t.Errorf("ColorEnum missing constant %v", want)
		}
	}
}

// ── Construction ─────────────────────────────────────────────────────

func TestNew_PanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("New() should panic on empty args")
		}
	}()
	enum.New[int]()
}

func TestNew_PanicsOnDuplicateName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("New() should panic on duplicate name")
		}
	}()
	enum.New(
		enum.ItemFrom("A", "A", 1),
		enum.ItemFrom("A", "A2", 2),
	)
}

func TestNew_PanicsOnDuplicateValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("New() should panic on duplicate value")
		}
	}()
	enum.New(
		enum.ItemFrom("A", "A", 1),
		enum.ItemFrom("B", "B", 1),
	)
}
