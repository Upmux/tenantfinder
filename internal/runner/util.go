package runner

import (
	"strings"

	"github.com/pkg/errors"
	stringsutil "github.com/projectdiscovery/utils/strings"
)

var (
	ErrEmptyInput = errors.New("empty data")
)

func sanitize(data string) (string, error) {
	data = strings.Trim(data, "\n\t\"'` ")
	if data == "" {
		return "", ErrEmptyInput
	}
	return data, nil
}

func preprocessDomain(s string) string {
	return stringsutil.NormalizeWithOptions(s,
		stringsutil.NormalizeOptions{
			StripComments: true,
			TrimCutset:    "\n\t\"'` ",
			Lowercase:     true,
		},
	)
}
