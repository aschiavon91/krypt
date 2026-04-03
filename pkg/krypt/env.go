package krypt

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

type entry struct {
	key   string
	value string
	raw   string // original line for comments/blank lines
}

func parseEnv(data []byte) []entry {
	var entries []entry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			entries = append(entries, entry{raw: line})
			continue
		}

		before, after, ok := strings.Cut(trimmed, "=")
		if !ok {
			entries = append(entries, entry{raw: line})
			continue
		}

		key := strings.TrimSpace(before)
		value := strings.TrimSpace(after)
		value = unquote(value)

		entries = append(entries, entry{key: key, value: value})
	}
	return entries
}

func unquote(s string) string {
	if len(s) >= 2 {
		quote := s[0]
		if (quote == '"' || quote == '\'') && s[len(s)-1] == quote {
			inner := s[1 : len(s)-1]
			// Only strip quotes if the inner content doesn't contain
			// the same unescaped quote character — avoids silent corruption
			// on values like 'don't' or "pass"word".
			if !strings.ContainsRune(inner, rune(quote)) {
				return inner
			}
		}
	}
	return s
}

func entriesToMap(entries []entry) map[string]string {
	m := make(map[string]string)
	for _, e := range entries {
		if e.key != "" {
			m[e.key] = e.value
		}
	}
	return m
}

func serializeEntries(entries []entry) []byte {
	var buf bytes.Buffer
	for _, e := range entries {
		if e.key != "" {
			fmt.Fprintf(&buf, "%s=%s\n", e.key, e.value)
		} else {
			buf.WriteString(e.raw)
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes()
}

func setEntry(entries []entry, key, value string) []entry {
	for i, e := range entries {
		if e.key == key {
			entries[i].value = value
			return entries
		}
	}
	return append(entries, entry{key: key, value: value})
}
