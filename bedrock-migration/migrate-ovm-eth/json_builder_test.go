package migrator

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestJSONBuilder(t *testing.T) {
	tests := []func(b *JSONBuilder){
		func(b *JSONBuilder) {
			b.Begin()
			b.SetString("test", "123")
			b.Next(true)
			b.Enter("object")
			b.SetString("a", "b")
			b.Next(false)
			b.Leave()
			b.Next(false)
			b.End()
		},
	}
	for i, tt := range tests {
		buf := new(bytes.Buffer)
		tt(NewJSONBuilder(buf))
		data, err := os.ReadFile(fmt.Sprintf("testdata/%d.json", i))
		require.NoError(t, err)
		require.Equal(t, string(data), buf.String())
	}
}
