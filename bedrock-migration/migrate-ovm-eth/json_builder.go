package migrator

import (
	"fmt"
	"io"
	"strings"
)

type JSONBuilder struct {
	level int
	w     io.Writer
}

func NewJSONBuilder(w io.Writer) *JSONBuilder {
	return &JSONBuilder{
		w: w,
	}
}

func (j *JSONBuilder) Begin() error {
	return j.write("{\n")
}

func (j *JSONBuilder) End() error {
	return j.write("}\n")
}

func (j *JSONBuilder) Enter(key string) error {
	j.level++
	return j.writeIndent(fmt.Sprintf("\"%s\": {\n", key), j.level)
}

func (j *JSONBuilder) Leave() error {
	if err := j.writeIndent("}", j.level); err != nil {
		return err
	}
	j.level--
	return nil
}

func (j *JSONBuilder) SetString(key, value string) error {
	return j.writeIndent(fmt.Sprintf("\"%s\": \"%s\"", key, value), j.level+1)
}

func (j *JSONBuilder) Next(comma bool) error {
	if comma {
		if err := j.write(","); err != nil {
			return err
		}
	}
	return j.write("\n")
}

func (j *JSONBuilder) writeIndent(value string, level int) error {
	if err := j.write(strings.Repeat(" ", level*2)); err != nil {
		return err
	}
	return j.write(value)
}

func (j *JSONBuilder) write(s string) error {
	_, err := j.w.Write([]byte(s))
	return err
}
