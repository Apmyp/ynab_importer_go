package formatter

import "testing"

func TestNewFormatter(t *testing.T) {
	f := NewFormatter()
	if f == nil {
		t.Error("NewFormatter() should return a non-nil Formatter")
	}
	if f.GetDateFormat() != "2006-01-02" {
		t.Errorf("NewFormatter() default date format = %q, want %q", f.GetDateFormat(), "2006-01-02")
	}
}

func TestSetDateFormat(t *testing.T) {
	f := NewFormatter()
	testFormat := "01/02/2006"
	f.SetDateFormat(testFormat)
	if f.GetDateFormat() != testFormat {
		t.Errorf("GetDateFormat() = %q, want %q", f.GetDateFormat(), testFormat)
	}
}

func TestGetDateFormat(t *testing.T) {
	f := NewFormatter()
	if f.GetDateFormat() == "" {
		t.Error("GetDateFormat() should return non-empty default")
	}
}
