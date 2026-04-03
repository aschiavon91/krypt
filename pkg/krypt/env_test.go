package krypt

import (
	"testing"
)

func TestParseEnvBasic(t *testing.T) {
	data := []byte("DATABASE_URL=postgres://localhost/test\nAPI_KEY=secret123\n")
	entries := parseEnv(data)

	m := entriesToMap(entries)
	if m["DATABASE_URL"] != "postgres://localhost/test" {
		t.Errorf("DATABASE_URL = %q", m["DATABASE_URL"])
	}
	if m["API_KEY"] != "secret123" {
		t.Errorf("API_KEY = %q", m["API_KEY"])
	}
}

func TestParseEnvEmptyValue(t *testing.T) {
	data := []byte("EMPTY_KEY=\n")
	entries := parseEnv(data)
	m := entriesToMap(entries)
	if v, ok := m["EMPTY_KEY"]; !ok || v != "" {
		t.Errorf("EMPTY_KEY = %q, ok = %v", v, ok)
	}
}

func TestParseEnvValueWithEquals(t *testing.T) {
	data := []byte("URL=postgres://host?opt=val\n")
	entries := parseEnv(data)
	m := entriesToMap(entries)
	if m["URL"] != "postgres://host?opt=val" {
		t.Errorf("URL = %q", m["URL"])
	}
}

func TestParseEnvDoubleQuotes(t *testing.T) {
	data := []byte(`KEY="value with spaces"` + "\n")
	entries := parseEnv(data)
	m := entriesToMap(entries)
	if m["KEY"] != "value with spaces" {
		t.Errorf("KEY = %q", m["KEY"])
	}
}

func TestParseEnvSingleQuotes(t *testing.T) {
	data := []byte(`KEY='literal value'` + "\n")
	entries := parseEnv(data)
	m := entriesToMap(entries)
	if m["KEY"] != "literal value" {
		t.Errorf("KEY = %q", m["KEY"])
	}
}

func TestParseEnvComments(t *testing.T) {
	data := []byte("# this is a comment\nKEY=value\n# another comment\n")
	entries := parseEnv(data)
	m := entriesToMap(entries)
	if len(m) != 1 {
		t.Errorf("expected 1 key, got %d", len(m))
	}
	if m["KEY"] != "value" {
		t.Errorf("KEY = %q", m["KEY"])
	}
}

func TestParseEnvBlankLines(t *testing.T) {
	data := []byte("KEY1=val1\n\nKEY2=val2\n\n")
	entries := parseEnv(data)
	m := entriesToMap(entries)
	if len(m) != 2 {
		t.Errorf("expected 2 keys, got %d", len(m))
	}
}

func TestRoundTrip(t *testing.T) {
	original := "# Database\nDATABASE_URL=postgres://localhost/test\n\n# API\nAPI_KEY=secret\n"
	entries := parseEnv([]byte(original))

	serialized := serializeEntries(entries)
	reparsed := parseEnv(serialized)

	m1 := entriesToMap(entries)
	m2 := entriesToMap(reparsed)

	if len(m1) != len(m2) {
		t.Fatalf("map sizes differ: %d vs %d", len(m1), len(m2))
	}
	for k, v := range m1 {
		if m2[k] != v {
			t.Errorf("key %s: %q vs %q", k, v, m2[k])
		}
	}
}

func TestSetEntryUpdate(t *testing.T) {
	entries := parseEnv([]byte("KEY1=old\nKEY2=keep\n"))
	entries = setEntry(entries, "KEY1", "new")

	m := entriesToMap(entries)
	if m["KEY1"] != "new" {
		t.Errorf("KEY1 = %q, want %q", m["KEY1"], "new")
	}
	if m["KEY2"] != "keep" {
		t.Errorf("KEY2 = %q, want %q", m["KEY2"], "keep")
	}
}

func TestSetEntryAppend(t *testing.T) {
	entries := parseEnv([]byte("KEY1=val1\n"))
	entries = setEntry(entries, "KEY2", "val2")

	m := entriesToMap(entries)
	if len(m) != 2 {
		t.Errorf("expected 2 keys, got %d", len(m))
	}
	if m["KEY2"] != "val2" {
		t.Errorf("KEY2 = %q", m["KEY2"])
	}
}

func TestEntriesToMap(t *testing.T) {
	entries := []entry{
		{key: "A", value: "1"},
		{raw: "# comment"},
		{key: "B", value: "2"},
		{raw: ""},
	}
	m := entriesToMap(entries)
	if len(m) != 2 {
		t.Errorf("expected 2, got %d", len(m))
	}
}
