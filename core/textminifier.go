package core

import (
	"io"
	"io/ioutil"
	"regexp"

	"github.com/tdewolff/minify/v2"
)

type TextMinifier struct{}

var TextMinify TextMinifier

func Minify(m *minify.M, w io.Writer, r io.Reader, params map[string]string) error {
	return TextMinify.Minify(m, w, r, params)
}

func (c *TextMinifier) Minify(m *minify.M, w io.Writer, r io.Reader, params map[string]string) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	rx := regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)
	s := rx.ReplaceAllString(string(b), "")
	_, err = w.Write([]byte(s))
	if err != nil {
		return err

	return nil
}