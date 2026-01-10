package parser

import "testing"

func TestNewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Error("NewParser() should return a non-nil Parser")
	}
	if p.GetDelimiter() != "," {
		t.Errorf("NewParser() default delimiter = %q, want %q", p.GetDelimiter(), ",")
	}
}

func TestSetDelimiter(t *testing.T) {
	p := NewParser()
	testDelim := ";"
	p.SetDelimiter(testDelim)
	if p.GetDelimiter() != testDelim {
		t.Errorf("GetDelimiter() = %q, want %q", p.GetDelimiter(), testDelim)
	}
}

func TestGetDelimiter(t *testing.T) {
	p := NewParser()
	if p.GetDelimiter() == "" {
		t.Error("GetDelimiter() should return non-empty default")
	}
}
