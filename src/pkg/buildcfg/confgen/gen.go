// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

func parseLine(s string) (d Define) {
	d = Define{
		Words: strings.Fields(s),
	}

	return
}

// Define is a struct that contains one line of configuration words.
type Define struct {
	Words []string
}

// WriteLine writes a line of configuration.
func (d Define) WriteLine() (s string) {
	s = "const " + d.Words[1] + " = " + d.Words[2]

	if len(d.Words) > 3 {
	}

	for _, w := range d.Words[3:] {
		s += " + " + w
	}
	return s
}

var confgenTemplate = template.Must(template.New("").Parse(`// Code generated by go generate; DO NOT EDIT.
package buildcfg
{{ range $i, $d := . }}
{{$d.WriteLine -}}
{{end}}
`))

func main() {
	outFile, err := os.Create("config.go")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer outFile.Close()

	inFile, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	header := []Define{}
	s := bufio.NewScanner(bytes.NewReader(inFile))
	for s.Scan() {
		header = append(header, parseLine(s.Text()))
	}

	confgenTemplate.Execute(outFile, header)
}
