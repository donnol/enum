package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/donnol/enum"
)

type Priority int

type Priorities struct {
	enum.Struct[Priority]
	Low    Priority `enum:"0,低"`
	Medium Priority `enum:"1,中"`
	High   Priority `enum:"2,高"`
}

func TestStruct_InitPopulatesValues(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	if p.Low != 0 || p.Medium != 1 || p.High != 2 {
		t.Errorf("fields = %d,%d,%d want 0,1,2", p.Low, p.Medium, p.High)
	}
}

func TestStruct_PointerEmbed(t *testing.T) {
	// Embedding *Struct[T] should work the same as embedding Struct[T].
	type PtrPriorities struct {
		*enum.Struct[Priority]
		Low    Priority `enum:"0,低"`
		Medium Priority `enum:"1,中"`
		High   Priority `enum:"2,高"`
	}
	p := enum.InitFor[Priority, PtrPriorities]()

	// Ensure the struct was populated correctly.
	if p.Low != 0 || p.Medium != 1 || p.High != 2 {
		t.Errorf("pointer-embed fields = %d,%d,%d want 0,1,2", p.Low, p.Medium, p.High)
	}
	item, ok := p.ByKey("Medium")
	if !ok || item.Value() != Priority(1) {
		t.Errorf("ByKey(Medium) = %v,%v want 1,true", item.Value(), ok)
	}
	if p.ToMap() == nil {
		t.Error("ToMap should work with pointer embed")
	}
	// JSON round-trip.
	b, err := json.Marshal(p.Enum())
	if err != nil || len(b) == 0 {
		t.Errorf("MarshalJSON with pointer embed: %v", err)
	}
}

func TestStruct_ByKey(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	item, ok := p.ByKey("Medium")
	if !ok || item.Value() != Priority(1) || item.Name() != "中" {
		t.Errorf("ByKey(Medium) = value=%v, name=%q, ok=%v want 1, 中, true", item.Value(), item.Name(), ok)
	}
}

func TestStruct_ByValue(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	item, ok := p.ByValue(Priority(2))
	if !ok || item.Key() != "High" || item.Name() != "高" {
		t.Errorf("ByValue(2) = %q/%q,%v want High/高,true", item.Key(), item.Name(), ok)
	}
}

func TestStruct_Contains(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	if !p.Contains(Priority(1)) {
		t.Error("Contains(1) = false, want true")
	}
	if p.Contains(Priority(99)) {
		t.Error("Contains(99) = true, want false")
	}
}

func TestStruct_Len(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	if p.Len() != 3 {
		t.Errorf("Len = %d, want 3", p.Len())
	}
}

func TestStruct_All(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	all := p.All()
	if len(all) != 3 {
		t.Fatalf("len = %d, want 3", len(all))
	}
	if all[0].Key() != "Low" || all[0].Name() != "低" {
		t.Errorf("first item = %q/%q want Low/低", all[0].Key(), all[0].Name())
	}
	if all[1].Key() != "Medium" || all[1].Name() != "中" {
		t.Errorf("second item = %q/%q want Medium/中", all[1].Key(), all[1].Name())
	}
	// Immutability.
	all[0] = enum.ItemFrom("x", "x", Priority(99))
	if p.All()[0].Key() != "Low" {
		t.Error("All() returned internal slice")
	}
}

func TestStruct_Index(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	item, ok := p.Index(1)
	if !ok || item.Key() != "Medium" || item.Name() != "中" {
		t.Errorf("Index(1) = %q/%q,%v want Medium/中,true", item.Key(), item.Name(), ok)
	}
	_, ok = p.Index(99)
	if ok {
		t.Error("Index(99) should not be found")
	}
}

func TestStruct_Range(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	count := 0
	for name, item := range p.Range() {
		if name == "" || item.Key() == "" {
			t.Error("empty name/item in Range")
		}
		if name != item.Key() {
			t.Errorf("name mismatch: %q != %q", name, item.Key())
		}
		if item.Name() == "" {
			t.Errorf("item %q has empty Name", item.Key())
		}
		count++
	}
	if count != 3 {
		t.Errorf("Range iterated %d, want 3", count)
	}
}

func TestStruct_Values(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	vals := p.Values()
	if len(vals) != 3 || vals[0] != 0 || vals[1] != 1 || vals[2] != 2 {
		t.Errorf("Values = %v, want [0 1 2]", vals)
	}
}

// ── string-based Struct ──────────────────────────────────────────────

type Severity string

type SeverityEnum struct {
	enum.Struct[Severity]
	Info  Severity `enum:",信息"`
	Warn  Severity `enum:"WARN,警告"`
	Error Severity `enum:",错误"`
}

func TestStruct_String_InitPopulatesValues(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()

	if s.Info != "INFO" || s.Warn != "WARN" || s.Error != "ERROR" {
		t.Errorf("fields = %q,%q,%q want INFO,WARN,ERROR", s.Info, s.Warn, s.Error)
	}
	// Verify auto-derived value matches the field name uppercased.
	if s.Info != Severity("INFO") {
		t.Error("auto-derived value should be field name uppercased")
	}
}

func TestStruct_String_ByKey(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()

	item, ok := s.ByKey("Warn")
	if !ok || item.Value() != Severity("WARN") || item.Name() != "警告" {
		t.Errorf("ByKey(Warn) = value=%v, name=%q, ok=%v want WARN, 警告, true",
			item.Value(), item.Name(), ok)
	}
}

func TestStruct_String_ByValue(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()

	item, ok := s.ByValue(Severity("ERROR"))
	if !ok || item.Key() != "Error" || item.Name() != "错误" {
		t.Errorf("ByValue(ERROR) = %q/%q,%v want Error/错误,true",
			item.Key(), item.Name(), ok)
	}
}

func TestStruct_String_Contains(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()

	if !s.Contains(Severity("WARN")) {
		t.Error("Contains(WARN) = false, want true")
	}
	if s.Contains(Severity("DEBUG")) {
		t.Error("Contains(DEBUG) = true, want false")
	}
}

func TestStruct_String_Index(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()

	item, ok := s.Index(2)
	if !ok || item.Key() != "Error" || item.Name() != "错误" || item.Value() != Severity("ERROR") {
		t.Errorf("Index(2) = %q/%q/%v,%v want Error/错误/ERROR,true",
			item.Key(), item.Name(), item.Value(), ok)
	}
	_, ok = s.Index(99)
	if ok {
		t.Error("Index(99) should not be found")
	}
}

func TestStruct_String_Range(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()

	count := 0
	for name, item := range s.Range() {
		if item.Name() == "" {
			t.Errorf("item %q has empty Name", item.Key())
		}
		if name != item.Key() {
			t.Errorf("name mismatch: %q != %q", name, item.Key())
		}
		count++
	}
	if count != 3 {
		t.Errorf("Range iterated %d, want 3", count)
	}
}

// ── switch on struct fields ──────────────────────────────────────────
// Struct fields are Go constants usable directly in switch statements.
// The enum provides metadata (names, iteration), but the switch
// branches on the field values for compile-time safety.

func stringForPriority(e Priorities, v Priority) string {
	switch v {
	case e.Low:
		return "handle low"
	case e.Medium:
		return "handle medium"
	case e.High:
		return "handle high"
	default:
		return "unknown"
	}
}

func TestStruct_Switch_Int(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	tests := []struct {
		val  Priority
		want string
	}{
		{p.Low, "handle low"},
		{p.Medium, "handle medium"},
		{p.High, "handle high"},
		{Priority(99), "unknown"},
	}
	for _, tt := range tests {
		got := stringForPriority(p, tt.val)
		if got != tt.want {
			t.Errorf("%v → %q, want %q", tt.val, got, tt.want)
		}
	}
}

func actionForSeverity(e SeverityEnum, v Severity) string {
	switch v {
	case e.Info:
		return "log"
	case e.Warn:
		return "alert"
	case e.Error:
		return "page"
	default:
		return "ignore"
	}
}

func TestStruct_Switch_String(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()

	if got := actionForSeverity(s, Severity("INFO")); got != "log" {
		t.Errorf("INFO → %q, want log", got)
	}
	if got := actionForSeverity(s, Severity("DEBUG")); got != "ignore" {
		t.Errorf("DEBUG → %q, want ignore", got)
	}
}

// Verify that field mutation does not affect the enum's internal data.
func TestStruct_EnumUnaffectedByFieldMutation(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	// Mutate the field — this only changes the struct field value for
	// local use (e.g. could break your switch), but does NOT corrupt
	// the enum's internal lookup maps.
	p.Medium = Priority(99)
	if p.Medium != Priority(99) {
		t.Error("field mutation not applied")
	}

	// Enum lookup is NOT affected — the internal map still holds 1 as
	// the value for "Medium".
	item, ok := p.ByValue(Priority(1))
	if !ok || item.Key() != "Medium" {
		t.Errorf("ByValue(1) after field mutation = %q,%v want Medium,true", item.Key(), ok)
	}
	// The mutated field value (99) is NOT known to the enum.
	if p.Contains(Priority(99)) {
		t.Error("enum should not contain the mutated value")
	}
}

type IntEnum struct {
	enum.Struct[int]
	A int `enum:"1,A"`
	B int `enum:"2,B"`
	C int `enum:"3,C"`
}

func TestStruct_PlainInt(t *testing.T) {
	e := enum.InitFor[int, IntEnum]()

	if e.A != 1 || e.B != 2 || e.C != 3 {
		t.Errorf("fields = %d,%d,%d want 1,2,3", e.A, e.B, e.C)
	}
	item, ok := e.ByValue(2)
	if !ok || item.Key() != "B" || item.Name() != "B" {
		t.Errorf("ByValue(2) = %q/%q,%v want B/B,true", item.Key(), item.Name(), ok)
	}
}

type UintEnum struct {
	enum.Struct[uint]
	A uint `enum:"1,A"`
	B uint `enum:"2,B"`
	C uint `enum:"3,C"`
}

func TestStruct_PlainUint(t *testing.T) {
	e := enum.InitFor[uint, UintEnum]()

	if e.A != 1 || e.B != 2 || e.C != 3 {
		t.Errorf("fields = %d,%d,%d want 1,2,3", e.A, e.B, e.C)
	}
	item, ok := e.ByValue(2)
	if !ok || item.Key() != "B" || item.Name() != "B" {
		t.Errorf("ByValue(2) = %q/%q,%v want B/B,true", item.Key(), item.Name(), ok)
	}
}

type IntEnumInvalid struct {
	enum.Struct[int]
	A int `enum:"1,A"`
	B int `enum:"2,B"`
	C int `enum:"c,C"`
}

func TestStruct_PlainIntInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("enum: cannot parse \"c\" as int: strconv.ParseInt: parsing \"c\": invalid syntax")
		}
	}()
	enum.InitFor[int, IntEnumInvalid]()
}

type UintEnumInvalid struct {
	enum.Struct[uint]
	A uint `enum:"1,A"`
	B uint `enum:"b,B"`
	C uint `enum:"3,C"`
}

func TestStruct_PlainUintInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("enum: cannot parse \"b\" as uint: strconv.ParseUint: parsing \"b\": invalid syntax")
		}
	}()
	enum.InitFor[uint, UintEnumInvalid]()
}

// ── plain string ─────────────────────────────────────────────────────

type StringEnum struct {
	enum.Struct[string]
	Red  string `enum:",红色"`
	Blue string `enum:",蓝色"`
}

func TestStruct_PlainString(t *testing.T) {
	e := enum.InitFor[string, StringEnum]()

	if e.Red != "RED" || e.Blue != "BLUE" {
		t.Errorf("fields = %q,%q want RED,BLUE", e.Red, e.Blue)
	}
	item, ok := e.ByKey("Blue")
	if !ok || item.Value() != "BLUE" || item.Name() != "蓝色" {
		t.Errorf("ByKey(Blue) = %q/%q,%v want BLUE/蓝色,true",
			item.Value(), item.Name(), ok)
	}
	// Contains.
	if !e.Contains("RED") || e.Contains("GREEN") {
		t.Errorf("Contains = %v/%v want true/false", e.Contains("RED"), e.Contains("GREEN"))
	}
}

// ── bad tag ─────────────────────────────────────────────────────

type EmptyEnumTag struct {
	enum.Struct[int]
	E int
	F int
	G int
}

func TestStruct_EmptyEnumTag(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("enum.Init: EmptyEnumTag.E missing enum tag")
		}
	}()
	enum.InitFor[int, EmptyEnumTag]()
}

type WrongEnumTag struct {
	enum.Struct[int]
	E int `enum:"1"`
	F int `enum:"2"`
	G int `enum:"3"`
}

func TestStruct_WrongEnumTag(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("enum.Init: WrongEnumTag.E invalid enum tag \"1\"")
		}
	}()
	enum.InitFor[int, WrongEnumTag]()
}

// ── duplicate detection ──────────────────────────────────────────────

type DuplicateValue struct {
	enum.Struct[int]
	A int `enum:"1,A"`
	B int `enum:"1,B"` // same value as A
}

type DuplicateName struct {
	enum.Struct[string]
	Red  string `enum:",红"`
	Also string `enum:"RED,红"` // same name
}

func TestStruct_PanicsOnDuplicateValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Init with duplicate values should panic")
		}
	}()
	enum.InitFor[int, DuplicateValue]()
}

func TestStruct_PanicsOnDuplicateName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Init with duplicate names should panic")
		}
	}()
	enum.InitFor[string, DuplicateName]()
}

// ── dot & slash values ─────────────────────────────────────────────

type Endpoint string

type Endpoints struct {
	enum.Struct[Endpoint]
	OrderAdd   Endpoint `enum:",订单创建"`
	OrderList  Endpoint `enum:",订单列表"`
	UserLogin  Endpoint `enum:"/user/login,用户登录"`
	KafkaTopic Endpoint `enum:"zapp.order.created,kafka主题"`
}

func TestStruct_DotAndSlashValues(t *testing.T) {
	e := enum.InitFor[Endpoint, Endpoints]()

	// Auto-derived from field name (uppercased).
	if e.OrderAdd != Endpoint("ORDERADD") {
		t.Errorf("OrderAdd = %q, want ORDERADD", e.OrderAdd)
	}
	// Explicit value with slash.
	if e.UserLogin != Endpoint("/user/login") {
		t.Errorf("UserLogin = %q, want /user/login", e.UserLogin)
	}
	// Explicit value with dot.
	if e.KafkaTopic != Endpoint("zapp.order.created") {
		t.Errorf("KafkaTopic = %q, want zapp.order.created", e.KafkaTopic)
	}

	// Lookup by value with dot notation.
	item, ok := e.ByValue(Endpoint("zapp.order.created"))
	if !ok || item.Name() != "kafka主题" {
		t.Errorf("ByValue(zapp.order.created) = %q,%v want kafka主题,true", item.Name(), ok)
	}
	// Lookup by value with slash.
	item, ok = e.ByValue(Endpoint("/user/login"))
	if !ok || item.Name() != "用户登录" {
		t.Errorf("ByValue(/user/login) = %q,%v want 用户登录,true", item.Name(), ok)
	}
	// Contains.
	if !e.Contains(Endpoint("/user/login")) {
		t.Error("Contains(/user/login) = false, want true")
	}
	if e.Contains(Endpoint("/user/delete")) {
		t.Error("Contains(/user/delete) = true, want false")
	}
}

// ── switch on dot/slash values ─────────────────────────────────────

func topicConfig(e Endpoints, v Endpoint) string {
	switch v {
	case e.KafkaTopic:
		return "handle order created"
	case e.UserLogin:
		return "handle login"
	default:
		return "unknown"
	}
}

func TestStruct_Switch_DotSlash(t *testing.T) {
	e := enum.InitFor[Endpoint, Endpoints]()

	if got := topicConfig(e, e.KafkaTopic); got != "handle order created" {
		t.Errorf("KafkaTopic → %q, want handle order created", got)
	}
	if got := topicConfig(e, e.UserLogin); got != "handle login" {
		t.Errorf("UserLogin → %q, want handle login", got)
	}
	if got := topicConfig(e, Endpoint("nope")); got != "unknown" {
		t.Errorf("unknown → %q, want unknown", got)
	}
}

// ── InitFor ───────────────────────────────────────────────────────────

func TestStruct_InitFor_Int(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()
	if p.Low != 0 || p.Medium != 1 || p.High != 2 {
		t.Errorf("fields = %d,%d,%d want 0,1,2", p.Low, p.Medium, p.High)
	}
	item, ok := p.ByKey("Medium")
	if !ok || item.Name() != "中" {
		t.Errorf("ByKey = %q,%v want 中,true", item.Name(), ok)
	}
	// Contains.
	if !p.Contains(Priority(2)) {
		t.Error("Contains(2) = false, want true")
	}
}

func TestStruct_InitFor_String(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()
	if s.Info != "INFO" || s.Warn != "WARN" {
		t.Errorf("fields = %q,%q want INFO,WARN", s.Info, s.Warn)
	}
	item, ok := s.ByKey("Warn")
	if !ok || item.Value() != Severity("WARN") {
		t.Errorf("ByKey = %q,%v want WARN,true", item.Value(), ok)
	}
}

func TestStruct_InitFor_PlainString(t *testing.T) {
	e := enum.InitFor[string, StringEnum]()
	if e.Red != "RED" || e.Blue != "BLUE" {
		t.Errorf("fields = %q,%q want RED,BLUE", e.Red, e.Blue)
	}
	if !e.Contains("RED") {
		t.Error("Contains(RED) = false, want true")
	}
}

func TestStruct_InitFor_DotSlash(t *testing.T) {
	e := enum.InitFor[Endpoint, Endpoints]()
	if e.UserLogin != Endpoint("/user/login") {
		t.Errorf("UserLogin = %q, want /user/login", e.UserLogin)
	}
	item, ok := e.ByValue(Endpoint("zapp.order.created"))
	if !ok || item.Name() != "kafka主题" {
		t.Errorf("ByValue = %q,%v want kafka主题,true", item.Name(), ok)
	}
}

// ── type mismatch detection ──────────────────────────────────────────

func TestStruct_InitFor_TypeMismatchPanics(t *testing.T) {
	// IntEnum has Struct[int], but T is string.
	defer func() {
		if r := recover(); r == nil {
			t.Error("InitFor[string, IntEnum] should panic — type mismatch")
		}
	}()
	enum.InitFor[string, IntEnum]()
}

// ── JSON serialization ────────────────────────────────────────────────

func TestStruct_MarshalJSON(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()
	b, err := json.Marshal(p.Enum())
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(b, &items); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d, want 3", len(items))
	}
	// Verify order and fields.
	if items[0]["key"] != "Low" || items[0]["name"] != "低" {
		t.Errorf("item[0] = %v, want {name:Low, name:低, value:0}", items[0])
	}
	if items[1]["key"] != "Medium" || items[1]["name"] != "中" {
		t.Errorf("item[1] = %v", items[1])
	}
	// Check value types.
	if v, ok := items[0]["value"].(float64); !ok || int(v) != 0 {
		t.Errorf("value = %v, want 0", items[0]["value"])
	}
}

func TestStruct_MarshalJSON_String(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()
	b, err := json.Marshal(s.Enum())
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(b, &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if items[0]["value"] != "INFO" || items[0]["name"] != "信息" {
		t.Errorf("item[0] = %v, want {name:Info, name:信息, value:INFO}", items[0])
	}
}

func TestStruct_MarshalJSON_eventTopics(t *testing.T) {
	e := enum.InitFor[Endpoint, Endpoints]()
	b, err := json.Marshal(e.Enum())
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var items []map[string]any
	json.Unmarshal(b, &items)
	if len(items) != 4 {
		t.Fatalf("len = %d, want 4", len(items))
	}
	if items[2]["value"] != "/user/login" {
		t.Errorf("item[2].value = %v, want /user/login", items[2]["value"])
	}
	if items[3]["value"] != "zapp.order.created" {
		t.Errorf("item[3].value = %v, want zapp.order.created", items[3]["value"])
	}
}

// ── JSON round-trip ─────────────────────────────────────────────────

func TestEnum_JSON_RoundTrip(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()

	// Marshal.
	b, err := json.Marshal(p.Enum())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Unmarshal into a new Enum.
	var e enum.Enum[Priority]
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify contents match.
	if e.Len() != p.Len() {
		t.Fatalf("Len = %d, want %d", e.Len(), p.Len())
	}
	item, ok := e.ByKey("Medium")
	if !ok || item.Name() != "中" || item.Value() != Priority(1) {
		t.Errorf("ByKey(Medium) = %q/%v/%v, want 中/1/true", item.Name(), item.Value(), ok)
	}
}

func TestEnum_JSON_UnmarshalInvalid(t *testing.T) {
	var e enum.Enum[Priority]
	// Duplicate name — should return error, not panic.
	err := json.Unmarshal([]byte(`[{"key":"A","name":"a","value":1},{"key":"A","name":"b","value":2}]`), &e)
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestEnum_JSON_UnmarshalString(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()
	b, _ := json.Marshal(s.Enum())
	var e enum.Enum[Severity]
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	item, ok := e.ByKey("Warn")
	if !ok || item.Name() != "警告" || item.Value() != Severity("WARN") {
		t.Errorf("ByKey(Warn) = %q/%v/%v, want 警告/WARN/true", item.Name(), item.Value(), ok)
	}
}

func TestEnum_JSON_RoundTrip_eventTopics(t *testing.T) {
	evts := enum.InitFor[Endpoint, Endpoints]()
	b, _ := json.Marshal(evts.Enum())
	var e enum.Enum[Endpoint]
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	item, ok := e.ByKey("KafkaTopic")
	if !ok || item.Value() != Endpoint("zapp.order.created") {
		t.Errorf("ByKey(KafkaTopic) = %q/%v, want zapp.order.created/true", item.Value(), ok)
	}
}

// ── disabled / ext scenarios ──────────────────────────────────────

func TestItemFrom_Disabled(t *testing.T) {
	item := enum.ItemFrom("Deprecated", "已废弃", 99, enum.WithDisabled[int]())
	if !item.IsDisabled() {
		t.Error("expected disabled")
	}
	if item.Key() != "Deprecated" || item.Value() != 99 {
		t.Error("other fields corrupted")
	}
}

func TestItemFrom_Ext(t *testing.T) {
	item := enum.ItemFrom("A", "选项A", 1,
		enum.WithExt[int](map[string]string{"css": "primary", "icon": "star"}),
	)
	ext := item.Ext()
	if ext["css"] != "primary" || ext["icon"] != "star" {
		t.Errorf("Ext = %v, want {css:primary, icon:star}", ext)
	}
	// Verify immutability of returned map.
	ext["css"] = "mutated"
	if item.Ext()["css"] != "primary" {
		t.Error("Ext() returned internal map reference")
	}
}

func TestEnum_AddExt(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()
	ext := p.GetExt("Low")
	if ext != nil {
		t.Errorf("GetExt(Low) = %q, want nil", ext)
	}

	p.AddExt("Low", "css", "muted")
	p.AddExt("Medium", "css", "primary")

	// add same ext
	p.AddExt("Medium", "css", "primary")
	// add same name but different kv
	p.AddExt("Medium", "color", "red")

	if ext := p.GetExt("Low"); ext["css"] != "muted" {
		t.Errorf("GetExt(Low).css = %q, want muted", ext["css"])
	}
	if ext := p.GetExt("Medium"); ext["css"] != "primary" {
		t.Errorf("GetExt(Low).css = %q, want primary", ext["css"])
	}
	if ext := p.GetExt("Medium"); ext["color"] != "red" {
		t.Errorf("GetExt(Low).css = %q, want red", ext["color"])
	}
	// Verify ByValue reflects the ext change (maps are updated).
	item, ok := p.ByValue(Priority(0))
	if !ok || item.Ext()["css"] != "muted" {
		t.Errorf("ByValue(0).Ext().css = %q, want muted", item.Ext()["css"])
	}
	// Unknown item name → panic (was silent no-op; a typo'd key would
	// otherwise hide a bug from the caller).
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("AddExt on unknown item should panic")
			}
		}()
		p.AddExt("NotFound", "x", "y")
	}()
}

func TestStruct_TagDisabled(t *testing.T) {
	type StatusWithDisabled struct {
		enum.Struct[int]
		Active     int `enum:"1,活跃"`
		Deprecated int `enum:"2,已废弃,disabled"`
	}
	s := enum.InitFor[int, StatusWithDisabled]()
	item, ok := s.ByValue(2)
	if !ok || !item.IsDisabled() {
		t.Errorf("Deprecated item should be disabled, got ok=%v disabled=%v", ok, item.IsDisabled())
	}
	// Active should NOT be disabled.
	item, ok = s.ByValue(1)
	if !ok || item.IsDisabled() {
		t.Errorf("Active item should not be disabled")
	}
}

func TestJSON_DisabledAndExt(t *testing.T) {
	type Mood string
	type Moods struct {
		enum.Struct[Mood]
		Happy Mood `enum:"HAPPY,开心"`
		Sad   Mood `enum:"SAD,难过,disabled"`
		Angry Mood `enum:"ANGRY,生气"`
	}
	m := enum.InitFor[Mood, Moods]()
	m.AddExt("Happy", "emoji", "😊")
	m.AddExt("Sad", "emoji", "😢")

	b, _ := json.Marshal(m.Enum())
	var items []map[string]any
	json.Unmarshal(b, &items)

	if items[1]["disabled"] != true {
		t.Error("Sad should have disabled:true in JSON")
	}
	ext, ok := items[0]["ext"].(map[string]any)
	if !ok || ext["emoji"] != "😊" {
		t.Errorf("Happy ext = %v, want {emoji:😊}", items[0]["ext"])
	}

	// Round-trip through JSON preserves disabled and ext.
	var e enum.Enum[Mood]
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	item, _ := e.ByKey("Sad")
	if !item.IsDisabled() || item.Ext()["emoji"] != "😢" {
		t.Errorf("round-trip Sad: disabled=%v ext=%v", item.IsDisabled(), item.Ext())
	}
}

// ── business scenario: order status ──────────────────────────────────

type OrderStatus int

type OrderStatuses struct {
	enum.Struct[OrderStatus]
	Pending    OrderStatus `enum:"0,待处理"`
	Processing OrderStatus `enum:"1,处理中"`
	Shipped    OrderStatus `enum:"2,已发货"`
	Cancelled  OrderStatus `enum:"3,已取消,disabled"` // legacy, display only
}

func TestScenario_OrderStatus(t *testing.T) {
	statuses := enum.InitFor[OrderStatus, OrderStatuses]()

	// Add metadata via ext.
	statuses.AddExt("Pending", "color", "orange")
	statuses.AddExt("Processing", "color", "blue")
	statuses.AddExt("Shipped", "color", "green")
	statuses.AddExt("Cancelled", "color", "gray")

	// Verify disabled.
	item, ok := statuses.ByValue(OrderStatus(3))
	if !ok || !item.IsDisabled() {
		t.Fatal("Cancelled should be disabled")
	}

	// Simulate frontend logic: filter out disabled items for dropdown.
	selectable := make([]string, 0)
	for _, it := range statuses.All() {
		if !it.IsDisabled() {
			selectable = append(selectable, it.Key())
		}
	}
	if len(selectable) != 3 {
		t.Errorf("selectable = %d, want 3", len(selectable))
	}

	// JSON for frontend.
	b, _ := json.Marshal(statuses.Enum())
	var frontend []map[string]any
	json.Unmarshal(b, &frontend)

	if frontend[3]["disabled"] != true {
		t.Error("Cancelled not marked disabled in JSON")
	}
	if ext, ok := frontend[0]["ext"].(map[string]any); !ok || ext["color"] != "orange" {
		t.Errorf("Pending ext = %v, want {color:orange}", frontend[0]["ext"])
	}
}

// ── field order ──────────────────────────────────────────────────────

func TestStruct_FieldOrderMatters(t *testing.T) {
	// Declared as: Medium, Low, High (reversed).
	type ReversedPriorities struct {
		enum.Struct[Priority]
		Medium Priority `enum:"1,中"`
		Low    Priority `enum:"0,低"`
		High   Priority `enum:"2,高"`
	}
	r := enum.InitFor[Priority, ReversedPriorities]()

	// Lookup by value — order doesn't matter here.
	item, ok := r.ByValue(Priority(0))
	if !ok || item.Key() != "Low" {
		t.Errorf("ByValue(0) = %q, want Low", item.Key())
	}

	// Iteration order follows declaration order.
	names := r.Keys()
	if names[0] != "Medium" || names[1] != "Low" || names[2] != "High" {
		t.Errorf("Names = %v, want [Medium Low High]", names)
	}

	// Index follows declaration order.
	item, ok = r.Index(0)
	if !ok || item.Key() != "Medium" {
		t.Errorf("Index(0) = %q, want Medium", item.Key())
	}

	// AddExt still works.
	r.AddExt("Low", "css", "muted")
	if r.GetExt("Low")["css"] != "muted" {
		t.Error("AddExt failed after reordering fields")
	}
}

// ── ToMap ────────────────────────────────────────────────────────────

func TestEnum_ToMap(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()
	m := p.ToMap()

	if m["Low"]["name"] != "低" || m["Low"]["value"].(Priority) != 0 {
		t.Errorf("Low = %v, want {name:低, value:0}", m["Low"])
	}
	if _, ok := m["Low"]["disabled"]; ok {
		t.Error("disabled should be omitted when false")
	}
	// Omitted key.
	if _, ok := m["NotFound"]; ok {
		t.Error("unknown key should be absent")
	}
	// Verify count.
	if len(m) != 3 {
		t.Errorf("len = %d, want 3", len(m))
	}
}

func TestEnum_ToMap_DisabledAndExt(t *testing.T) {
	type Mood string
	type Moods struct {
		enum.Struct[Mood]
		Happy Mood `enum:"HAPPY,开心"`
		Sad   Mood `enum:"SAD,难过,disabled"`
		Angry Mood `enum:"ANGRY,生气"`
	}
	ms := enum.InitFor[Mood, Moods]()
	ms.AddExt("Happy", "emoji", "😊")

	m := ms.ToMap()
	if m["Sad"]["disabled"] != true {
		t.Error("Sad should be disabled=true")
	}
	if ext, ok := m["Happy"]["ext"].(map[string]string); !ok || ext["emoji"] != "😊" {
		t.Errorf("Happy.ext = %v, want {emoji:😊}", m["Happy"]["ext"])
	}
}

func TestEnum_ToMap_JSON(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()
	b, err := json.Marshal(p.ToMap())
	if err != nil {
		t.Fatalf("marshal ToMap: %v", err)
	}
	var m map[string]map[string]any
	json.Unmarshal(b, &m)
	if m["Low"]["value"].(float64) != 0 {
		t.Errorf("Low.value = %v, want 0", m["Low"]["value"])
	}
}

func TestEnum_ToMap_EventTypes(t *testing.T) {
	e := enum.InitFor[Endpoint, Endpoints]()
	m := e.ToMap()

	s, ok := m["UserLogin"]["value"].(Endpoint)
	if !ok || s != Endpoint("/user/login") {
		t.Errorf("UserLogin = %v, want /user/login", m["UserLogin"]["value"])
	}
	s, ok = m["KafkaTopic"]["value"].(Endpoint)
	if !ok || s != Endpoint("zapp.order.created") {
		t.Errorf("KafkaTopic = %v, want zapp.order.created", m["KafkaTopic"]["value"])
	}
}

// ── multi-level dropdown via ext ────────────────────────────────────

type Category string

type Categories struct {
	enum.Struct[Category]
	Fruit     Category `enum:"FRUIT,水果"`
	Vegetable Category `enum:"VEGETABLE,蔬菜"`
}

type Product string

type Products struct {
	enum.Struct[Product]
	Apple  Product `enum:",苹果"`
	Banana Product `enum:",香蕉"`
	Carrot Product `enum:",胡萝卜"`
}

func TestScenario_MultiLevelDropdown(t *testing.T) {
	categories := enum.InitFor[Category, Categories]()
	products := enum.InitFor[Product, Products]()

	// Link products to categories via ext.
	products.AddExt("Apple", enum.ParentExtKey, "FRUIT")
	products.AddExt("Banana", enum.ParentExtKey, "FRUIT")
	products.AddExt("Carrot", enum.ParentExtKey, "VEGETABLE")

	// Filter products by parent.
	fruitProducts := filterByExtValue(products.All(), enum.ParentExtKey, "FRUIT")
	if len(fruitProducts) != 2 {
		t.Fatalf("fruit products = %d, want 2", len(fruitProducts))
	}
	if fruitProducts[0].Key() != "Apple" || fruitProducts[1].Key() != "Banana" {
		t.Errorf("fruit products = %v, want [Apple Banana]", fruitProducts)
	}

	// Serialize categories and products for frontend.
	type dropdownOption struct {
		Key      string           `json:"key"`
		Name     string           `json:"name"`
		Value    string           `json:"value"`
		Children []dropdownOption `json:"children,omitempty"`
	}

	// Build tree: categories → filtered children.
	var options []dropdownOption
	for _, cat := range categories.All() {
		children := filterByExtValue(products.All(), enum.ParentExtKey, string(cat.Value()))
		childOpts := make([]dropdownOption, len(children))
		for j, child := range children {
			childOpts[j] = dropdownOption{
				Key:   child.Key(),
				Name:  child.Key(),
				Value: string(child.Value()),
			}
		}
		options = append(options, dropdownOption{
			Key:      cat.Key(),
			Name:     cat.Name(),
			Value:    string(cat.Value()),
			Children: childOpts,
		})
	}

	b, _ := json.Marshal(options)
	var tree []map[string]any
	json.Unmarshal(b, &tree)

	if tree[0]["key"] != "Fruit" || len(tree[0]["children"].([]any)) != 2 {
		t.Errorf("fruit node = %v, want 2 children", tree[0])
	}
	if tree[1]["key"] != "Vegetable" || len(tree[1]["children"].([]any)) != 1 {
		t.Errorf("vegetable node = %v, want 1 child", tree[1])
	}
}

// ── recursive dropdown tree ──────────────────────────────────────────

// filterByExtValue filters items whose Ext map has the given key-value pair.
func filterByExtValue[T enum.EnumBase](items []enum.Item[T], key, value string) []enum.Item[T] {
	var out []enum.Item[T]
	for _, it := range items {
		if v, ok := it.Ext()[key]; ok && v == value {
			out = append(out, it)
		}
	}
	return out
}

// ── defensive checks in initStruct ───────────────────────────────────

func TestInitStruct_SkipsUnexportedAnonymousFields(t *testing.T) {
	// Embed an unexported base type with its own Struct[T] alongside an
	// exported Struct[T]. The unexported field is skipped (CanInterface
	// returns false), and the exported one is found.
	type ignored struct {
		enum.Struct[string] // unreachable from outer structs reflection
	}
	type wrapper struct {
		ignored                    // unexported anonymous field — skipped by CanInterface
		enum.Struct[string]        // exported — found after skip
		A                   string `enum:"1,甲"`
		B                   string `enum:"2,乙"`
	}

	w := enum.InitFor[string, wrapper]()

	item, ok := w.ByKey("A")
	if !ok || item.Value() != "1" {
		t.Fatalf("ByKey(A) = %v,%v want 1,true", item.Value(), ok)
	}
	if w.Len() != 2 {
		t.Errorf("Len = %d, want 2", w.Len())
	}
}

func TestInitStruct_SkipsUnexportedPointerField(t *testing.T) {
	// Embed an unexported *hidden type. initStruct should skip the
	// unexported pointer field (CanInterface and CanSet both fail)
	// and find the exported Struct[int] next.
	type hidden struct {
		enum.Struct[int]
		Alpha int `enum:"0,Alpha"`
	}
	type extended struct {
		*hidden              // unexported pointer — skipped
		enum.Struct[int]     // exported value — found
		Beta             int `enum:"1,Beta"`
		Gamma            int `enum:"2,Gamma"`
	}

	e := enum.InitFor[int, extended]()

	if e.Len() != 2 {
		t.Fatalf("Len = %d, want 2", e.Len())
	}
	item, ok := e.ByKey("Beta")
	if !ok || item.Value() != 1 {
		t.Errorf("ByKey(Beta) = %v,%v want 1,true", item.Value(), ok)
	}
	if e.Contains(2) {
		if !e.Contains(2) {
			t.Error("should contain Gamma")
		}
	}
}

func TestInitStruct_UnexportedFieldWithOtherExportedEmbed(t *testing.T) {
	// Same struct, two anonymous fields: one unexported, one exported.
	// The exported one should be found after skipping the unexported.
	type hidden struct {
		enum.Struct[int]
		X int `enum:"-1,hidden_x"`
	}
	type dual struct {
		hidden               // unexported — skipped
		enum.Struct[int]     // exported value embed — found
		Visible          int `enum:"10,visible"`
	}

	d := enum.InitFor[int, dual]()

	if d.Len() != 1 {
		t.Fatalf("Len = %d, want 1", d.Len())
	}
	item, ok := d.ByKey("Visible")
	if !ok || item.Value() != 10 {
		t.Errorf("Visible = %v,%v want 10,true", item.Value(), ok)
	}
}

// ── Tree / TreeOptions ─────────────────────────────────────────────
type eventTopic string

func TestStruct_Tree_FlatEnum(t *testing.T) {
	// Flat enum (no nested structs) → Tree returns leaf nodes.
	type Flats struct {
		enum.Struct[Priority]
		Low    Priority `enum:"0,低"`
		Medium Priority `enum:"1,中"`
	}
	e := enum.InitFor[Priority, Flats]()
	tree := e.Tree()

	if len(tree) != 2 {
		t.Fatalf("flat tree len = %d, want 2", len(tree))
	}
	if tree[0].Name != "低" || tree[0].Value != "0" {
		t.Errorf("tree[0] = {%s, %s}, want {低, 0}", tree[0].Name, tree[0].Value)
	}
	if tree[1].Name != "中" || tree[1].Value != "1" {
		t.Errorf("tree[1] = {%s, %s}, want {中, 1}", tree[1].Name, tree[1].Value)
	}
	if len(tree[0].Children) != 0 {
		t.Error("flat items should have no children")
	}
}

func TestStruct_TreeOptions_FlatEnum(t *testing.T) {
	type Flats struct {
		enum.Struct[Priority]
		Low  Priority `enum:"0,低"`
		High Priority `enum:"2,高,disabled"`
	}
	e := enum.InitFor[Priority, Flats]()
	opts := e.TreeOptions()

	if len(opts) != 2 {
		t.Fatalf("TreeOptions len = %d, want 2", len(opts))
	}
	if opts[0].Label != "低" || opts[0].Value != "0" {
		t.Errorf("opts[0] = {%s, %s}", opts[0].Label, opts[0].Value)
	}
	if !opts[1].Disabled {
		t.Error("disabled flag not propagated")
	}
}

func TestStruct_Tree_NestedOneLevel(t *testing.T) {
	type Nested struct {
		enum.Struct[eventTopic]
		UserManage struct {
			enum.Struct[eventTopic]
			Self     eventTopic `enum:"userm,用户管理"`
			UserList eventTopic `enum:"/user/list,用户列表"`
			UserAdd  eventTopic `enum:"/user/add,用户新增"`
		}
	}
	e := enum.InitFor[eventTopic, Nested]()
	tree := e.Tree()

	if len(tree) != 1 {
		t.Fatalf("tree len = %d, want 1", len(tree))
	}
	root := tree[0]
	if root.Name != "用户管理" || root.Value != "userm" {
		t.Errorf("root = {%s, %s}, want {用户管理, userm}", root.Name, root.Value)
	}
	if len(root.Children) != 2 {
		t.Fatalf("children = %d, want 2", len(root.Children))
	}
	if root.Children[0].Name != "用户列表" {
		t.Errorf("child[0] = %s, want 用户列表", root.Children[0].Name)
	}
	if root.Children[1].Name != "用户新增" {
		t.Errorf("child[1] = %s, want 用户新增", root.Children[1].Name)
	}
}

func TestStruct_Tree_NestedMultiLevel(t *testing.T) {
	type Deep struct {
		enum.Struct[eventTopic]
		China struct {
			enum.Struct[eventTopic]
			Self eventTopic `enum:"100,中国"`
			East struct {
				enum.Struct[eventTopic]
				Self     eventTopic `enum:"101,华东"`
				Shanghai eventTopic `enum:"10101,上海"`
			}
		}
	}
	e := enum.InitFor[eventTopic, Deep]()
	tree := e.Tree()

	if len(tree) != 1 {
		t.Fatalf("tree len = %d, want 1", len(tree))
	}
	china := tree[0]
	if china.Name != "中国" {
		t.Errorf("root = %s, want 中国", china.Name)
	}
	if len(china.Children) != 1 {
		t.Fatalf("china children = %d, want 1", len(china.Children))
	}
	east := china.Children[0]
	if east.Name != "华东" {
		t.Errorf("east = %s, want 华东", east.Name)
	}
	if len(east.Children) != 1 {
		t.Fatalf("east children = %d, want 1", len(east.Children))
	}
	if east.Children[0].Name != "上海" {
		t.Errorf("shanghai = %s, want 上海", east.Children[0].Name)
	}
}

func TestStruct_Tree_NestedPlusFlat(t *testing.T) {
	// Mix: nested struct + flat top-level items.
	type Mixed struct {
		enum.Struct[eventTopic]
		UserManage struct {
			enum.Struct[eventTopic]
			Self     eventTopic `enum:"userm,用户管理"`
			UserList eventTopic `enum:"/user/list,用户列表"`
		}
		UserCreated eventTopic `enum:"user.created,用户创建"`
	}
	e := enum.InitFor[eventTopic, Mixed]()
	tree := e.Tree()

	if len(tree) != 2 {
		t.Fatalf("tree len = %d, want 2", len(tree))
	}
	// Order should be: UserManage (nested struct) then UserCreated (flat).
	if tree[0].Name != "用户管理" {
		t.Errorf("tree[0] = %s, want 用户管理", tree[0].Name)
	}
	if tree[1].Name != "用户创建" {
		t.Errorf("tree[1] = %s, want 用户创建", tree[1].Name)
	}
}

func TestStruct_TreeOptions_NestedOneLevel(t *testing.T) {
	type Nested struct {
		enum.Struct[eventTopic]
		UserManage struct {
			enum.Struct[eventTopic]
			Self     eventTopic `enum:"userm,用户管理"`
			UserList eventTopic `enum:"/user/list,用户列表"`
		}
	}
	e := enum.InitFor[eventTopic, Nested]()
	opts := e.TreeOptions()

	if len(opts) != 1 {
		t.Fatalf("TreeOptions len = %d, want 1", len(opts))
	}
	if opts[0].Label != "用户管理" {
		t.Errorf("label = %s, want 用户管理", opts[0].Label)
	}
	if opts[0].Value != "userm" {
		t.Errorf("value = %s, want userm", opts[0].Value)
	}
	if len(opts[0].Children) != 1 {
		t.Fatalf("children = %d, want 1", len(opts[0].Children))
	}
	if opts[0].Children[0].Label != "用户列表" {
		t.Errorf("child label = %s, want 用户列表", opts[0].Children[0].Label)
	}
}

func TestStruct_All_ContainsNestedItems(t *testing.T) {
	// All() should return flat items including nested ones.
	type Mixed struct {
		enum.Struct[eventTopic]
		UserManage struct {
			enum.Struct[eventTopic]
			Self     eventTopic `enum:"userm,用户管理"`
			UserList eventTopic `enum:"/user/list,用户列表"`
		}
		UserCreated eventTopic `enum:"user.created,用户创建"`
	}
	e := enum.InitFor[eventTopic, Mixed]()

	all := e.All()
	if len(all) != 3 {
		t.Fatalf("All len = %d, want 3", len(all))
	}
	if e.Len() != 3 {
		t.Errorf("Len = %d, want 3", e.Len())
	}
	// Verify ByValue works for nested items too.
	if _, ok := e.ByValue(eventTopic("userm")); !ok {
		t.Error("ByValue(userm) should find nested item")
	}
}

func TestStruct_Tree_JSON(t *testing.T) {
	type Nested struct {
		enum.Struct[eventTopic]
		UserManage struct {
			enum.Struct[eventTopic]
			Self     eventTopic `enum:"userm,用户管理"`
			UserList eventTopic `enum:"/user/list,用户列表"`
		}
	}
	e := enum.InitFor[eventTopic, Nested]()
	b, err := json.Marshal(e.Tree())
	if err != nil {
		t.Fatal(err)
	}
	var nodes []enum.TreeNode
	if err := json.Unmarshal(b, &nodes); err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].Name != "用户管理" {
		t.Error("Tree JSON round-trip failed")
	}
}

func TestStruct_TreeOptions_JSON(t *testing.T) {
	type Nested struct {
		enum.Struct[eventTopic]
		UserManage struct {
			enum.Struct[eventTopic]
			Self     eventTopic `enum:"userm,用户管理"`
			UserList eventTopic `enum:"/user/list,用户列表"`
		}
	}
	e := enum.InitFor[eventTopic, Nested]()
	b, err := json.Marshal(e.TreeOptions())
	if err != nil {
		t.Fatal(err)
	}
	var opts []enum.CascaderOption
	if err := json.Unmarshal(b, &opts); err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 || opts[0].Label != "用户管理" {
		t.Error("TreeOptions JSON round-trip failed")
	}
}

func TestStruct_Tree_MatchesEventEmbedsPattern(t *testing.T) {
	// Match the EventEmbeds pattern from event.go:
	//   UserManage struct {
	//       Self     eventTopic `enum:"userm,用户管理"`
	//       Children struct {
	//           UserList eventTopic `enum:"/user/list,用户列表"`
	//       }
	//   }
	// Self is the tree root; Children items are nested under it.
	type Nested struct {
		enum.Struct[eventTopic]
		UserManage struct {
			enum.Struct[eventTopic]
			Self     eventTopic `enum:"userm,用户管理"`
			Children struct {
				enum.Struct[eventTopic]
				UserList eventTopic `enum:"/user/list,用户列表"`
				UserAdd  eventTopic `enum:"/user/add,用户新增"`
			}
		}
	}
	e := enum.InitFor[eventTopic, Nested]()

	// === value ===

	if e.UserManage.Self != "userm" {
		t.Errorf("e.UserManage.Self = %s, want userm", e.UserManage.Self)
	}
	if e.UserManage.Children.UserList != "/user/list" {
		t.Errorf("e.UserManage.Children.UserList = %s, want /user/list", e.UserManage.Children.UserList)
	}
	if e.UserManage.Children.UserAdd != "/user/add" {
		t.Errorf("e.UserManage.Children.UserAdd = %s, want /user/add", e.UserManage.Children.UserAdd)
	}

	// === by value ===

	if v, want := e.MustByValue(e.UserManage.Self), "UserManage.Self"; v.Key() != want || v.Value() != "userm" || v.Name() != "用户管理" {
		t.Errorf("UserManage.value = %s, want %s, item: %+v", v.Key(), want, v)
	}
	if v, want := e.MustByValue(e.UserManage.Children.UserList), "UserManage.Children.UserList"; v.Key() != want || v.Value() != "/user/list" || v.Name() != "用户列表" {
		t.Errorf("UserManage.value = %s, want %s, item: %+v", v.Key(), want, v)
	}
	if v, want := e.MustByValue(e.UserManage.Children.UserAdd), "UserManage.Children.UserAdd"; v.Key() != want || v.Value() != "/user/add" || v.Name() != "用户新增" {
		t.Errorf("UserManage.value = %s, want %s, item: %+v", v.Key(), want, v)
	}

	// === tree ===

	tree := e.Tree()

	if len(tree) != 1 {
		t.Fatalf("tree len = %d, want 1", len(tree))
	}
	root := tree[0]
	if root.Name != "用户管理" {
		t.Errorf("root = %s, want 用户管理", root.Name)
	}
	if len(root.Children) != 2 {
		t.Fatalf("children = %d, want 2 (UserList + UserAdd)", len(root.Children))
	}
	if root.Children[0].Name != "用户列表" {
		t.Errorf("child[0] = %s, want 用户列表", root.Children[0].Name)
	}
	if root.Children[1].Name != "用户新增" {
		t.Errorf("child[1] = %s, want 用户新增", root.Children[1].Name)
	}
}
