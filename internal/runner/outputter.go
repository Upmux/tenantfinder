package runner

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"

	"github.com/upmux/tenantfinder/pkg/resolve"
)

// OutputWriter outputs content to writers.
type OutputWriter struct {
	JSON bool
}

type jsonSourceResult struct {
	Domain string `json:"domain"`
	Input  string `json:"input"`
	Source string `json:"source"`
}

type jsonSourceIPResult struct {
	Domain string `json:"domain"`
	Input  string `json:"input"`
	Source string `json:"source"`
}

type jsonSourcesResult struct {
	Domain  string   `json:"domain"`
	Input   string   `json:"input"`
	Sources []string `json:"sources"`
}

// NewOutputWriter creates a new OutputWriter
func NewOutputWriter(json bool) *OutputWriter {
	return &OutputWriter{JSON: json}
}

func (o *OutputWriter) createFile(filename string, appendToFile bool) (*os.File, error) {
	if filename == "" {
		return nil, errors.New("empty filename")
	}

	dir := filepath.Dir(filename)

	if dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return nil, err
			}
		}
	}

	var file *os.File
	var err error
	if appendToFile {
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(filename)
	}
	if err != nil {
		return nil, err
	}

	return file, nil
}

// WriteHostIP writes the output list of domain to an io.Writer
func (o *OutputWriter) WriteHostIP(input string, results map[string]resolve.Result, writer io.Writer) error {
	var err error
	if o.JSON {
		err = writeJSONHostIP(input, results, writer)
	} else {
		err = writePlainHostIP(input, results, writer)
	}
	return err
}

func writePlainHostIP(_ string, results map[string]resolve.Result, writer io.Writer) error {
	bufwriter := bufio.NewWriter(writer)
	sb := &strings.Builder{}

	for _, result := range results {
		sb.WriteString(result.Host)
		sb.WriteString(",")
		sb.WriteString(result.IP)
		sb.WriteString(",")
		sb.WriteString(result.Source)
		sb.WriteString("\n")

		_, err := bufwriter.WriteString(sb.String())
		if err != nil {
			bufwriter.Flush()
			return err
		}
		sb.Reset()
	}
	return bufwriter.Flush()
}

func writeJSONHostIP(input string, results map[string]resolve.Result, writer io.Writer) error {
	encoder := jsoniter.NewEncoder(writer)

	var data jsonSourceIPResult

	for _, result := range results {
		data.Domain = result.Host
		data.Input = input
		data.Source = result.Source

		err := encoder.Encode(&data)
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteHost writes the output list of domain to an io.Writer
func (o *OutputWriter) WriteHost(input string, results map[string]resolve.HostEntry, writer io.Writer) error {
	var err error
	if o.JSON {
		err = writeJSONHost(input, results, writer)
	} else {
		err = writePlainHost(input, results, writer)
	}
	return err
}

func writePlainHost(_ string, results map[string]resolve.HostEntry, writer io.Writer) error {
	bufwriter := bufio.NewWriter(writer)
	sb := &strings.Builder{}

	for _, result := range results {
		sb.WriteString(result.Host)
		sb.WriteString("\n")

		_, err := bufwriter.WriteString(sb.String())
		if err != nil {
			bufwriter.Flush()
			return err
		}
		sb.Reset()
	}
	return bufwriter.Flush()
}

func writeJSONHost(input string, results map[string]resolve.HostEntry, writer io.Writer) error {
	encoder := jsoniter.NewEncoder(writer)

	var data jsonSourceResult
	for _, result := range results {
		data.Domain = result.Host
		data.Input = input
		data.Source = result.Source
		err := encoder.Encode(data)
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteSourceHost writes the output list of domain to an io.Writer
func (o *OutputWriter) WriteSourceHost(input string, sourceMap map[string]map[string]struct{}, writer io.Writer) error {
	var err error
	if o.JSON {
		err = writeSourceJSONHost(input, sourceMap, writer)
	} else {
		err = writeSourcePlainHost(input, sourceMap, writer)
	}
	return err
}

func writeSourceJSONHost(input string, sourceMap map[string]map[string]struct{}, writer io.Writer) error {
	encoder := jsoniter.NewEncoder(writer)

	var data jsonSourcesResult

	for host, sources := range sourceMap {
		data.Domain = host
		data.Input = input
		keys := make([]string, 0, len(sources))
		for source := range sources {
			keys = append(keys, source)
		}
		data.Sources = keys

		err := encoder.Encode(&data)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeSourcePlainHost(_ string, sourceMap map[string]map[string]struct{}, writer io.Writer) error {
	bufwriter := bufio.NewWriter(writer)
	sb := &strings.Builder{}

	for host, sources := range sourceMap {
		sb.WriteString(host)
		sb.WriteString(",[")
		sourcesString := ""
		for source := range sources {
			sourcesString += source + ","
		}
		sb.WriteString(strings.Trim(sourcesString, ", "))
		sb.WriteString("]\n")

		_, err := bufwriter.WriteString(sb.String())
		if err != nil {
			bufwriter.Flush()
			return err
		}
		sb.Reset()
	}
	return bufwriter.Flush()
}
