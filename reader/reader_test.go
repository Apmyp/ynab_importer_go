package reader

import "testing"

func TestNewReader(t *testing.T) {
	r := NewReader()
	if r == nil {
		t.Error("NewReader() should return a non-nil Reader")
	}
}

func TestSetPath(t *testing.T) {
	r := NewReader()
	testPath := "/test/messages"
	r.SetPath(testPath)
	if r.GetPath() != testPath {
		t.Errorf("GetPath() = %q, want %q", r.GetPath(), testPath)
	}
}

func TestGetPath(t *testing.T) {
	r := NewReader()
	if r.GetPath() != "" {
		t.Errorf("GetPath() on new Reader = %q, want empty string", r.GetPath())
	}
}
