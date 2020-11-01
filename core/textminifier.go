package core

import (
	"io"
	"io/ioutil"
	"regexp"

	"github.com/tdewolff/minify/v2"
)

// TextMinifier struct
type TextMinifier struct{}

// TextMinify TextMinifier
var TextMinify TextMinifier

// Minify ()
func Minify(m *minify.M, w io.Writer, r io.Reader, params map[string]string) error {
	return TextMinify.Minify(m, w, r, params)
}

// Minify (m, w, r, params)
// see https://github.com/tdewolff/minify/blob/master/minify.go for details
func (c *TextMinifier) Minify(_ *minify.M, w io.Writer, r io.Reader, _ map[string]string) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	rx := regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)
	s := rx.ReplaceAllString(string(b), "")
	_, err = w.Write([]byte(s))
	if err != nil {
		return err
	}

	return nil
}
