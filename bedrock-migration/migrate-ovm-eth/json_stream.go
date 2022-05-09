package migrator

import "io"

type JSONStream struct {
	level int
	w io.Writer
}

func (j *JSONStream) Object(key string)