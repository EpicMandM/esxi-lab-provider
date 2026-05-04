package logger

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	l := New()
	require.NotNil(t, l)
	assert.NotNil(t, l.writer)
}

func TestNewWithWriter(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf)
	require.NotNil(t, l)
	l.Info("hello")
	assert.Contains(t, buf.String(), "LEVEL=INFO")
	assert.Contains(t, buf.String(), "MESSAGE=hello")
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf)
	l.Info("test message", F("key", "value"))
	output := buf.String()
	assert.Contains(t, output, "LEVEL=INFO")
	assert.Contains(t, output, "MESSAGE=test message")
	assert.Contains(t, output, "key=value")
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf)
	l.Error("something broke", F("code", 500))
	output := buf.String()
	assert.Contains(t, output, "LEVEL=ERROR")
	assert.Contains(t, output, "MESSAGE=something broke")
	assert.Contains(t, output, "code=500")
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf)
	l.Warn("watch out")
	assert.Contains(t, buf.String(), "LEVEL=WARNING")
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf)
	l.Debug("details")
	assert.Contains(t, buf.String(), "LEVEL=DEBUG")
}

func TestLogMultipleFields(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf)
	l.Info("multi", F("a", 1), F("b", "two"))
	output := buf.String()
	assert.Contains(t, output, "a=1")
	assert.Contains(t, output, "b=two")
}

func TestFieldConstructors(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		key   string
	}{
		{"Action", Action("do"), "ACTION"},
		{"Status", Status("ok"), "STATUS"},
		{"VM", VM("vm1"), "VM"},
		{"User", User("alice"), "USER"},
		{"Count", Count(5), "COUNT"},
		{"Error", Error(errors.New("oops")), "ERROR"},
		{"Snapshot", Snapshot("snap1"), "SNAPSHOT"},
		{"Password", Password("secret"), "PASSWORD"},
		{"VMIndex", VMIndex(2), "VM_INDEX"},
		{"Events", Events(3), "EVENTS"},
		{"Restored", Restored(4), "RESTORED"},
		{"Failed", Failed(1), "FAILED"},
		{"TimeWindow", TimeWindow("±5min"), "TIME_WINDOW"},
		{"Reason", Reason("because"), "REASON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.key, tt.field.Key)
			assert.NotNil(t, tt.field.Value)
		})
	}
}

func TestF(t *testing.T) {
	f := F("mykey", 42)
	assert.Equal(t, "mykey", f.Key)
	assert.Equal(t, 42, f.Value)
}

func TestLogNoFields(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&buf)
	l.Info("no fields")
	output := buf.String()
	assert.Equal(t, "LEVEL=INFO MESSAGE=no fields\n", output)
}
