package runner

import (
	"bufio"
	"context"
	"io"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/upmux/tenantfinder/pkg/agent"

	"github.com/projectdiscovery/gologger"
	contextutil "github.com/projectdiscovery/utils/context"
	mapsutil "github.com/projectdiscovery/utils/maps"
)

// Runner is an instance of the subdomain enumeration
// client used to orchestrate the whole process.
type Runner struct {
	options   *Options
	agent     *agent.Agent
	rateLimit *agent.CustomRateLimit
	// TODO Add DNS resolution
	// resolverClient *resolve.Resolver
}

// NewRunner creates a new runner struct instance by parsing
// the configuration options, configuring sources, reading lists
// and setting up loggers, etc.
func NewRunner(options *Options) (*Runner, error) {
	options.configureOutput()
	runner := &Runner{options: options}

	// Check if the application loading with any provider configuration, then take it
	// Otherwise load the default provider config
	// if fileutil.FileExists(options.ProviderConfig) {
	// 	gologger.Info().Msgf("Loading provider config from %s", options.ProviderConfig)
	// 	options.loadProvidersFrom(options.ProviderConfig)
	// } else {
	// 	gologger.Info().Msgf("Loading provider config from the default location: %s", defaultProviderConfigLocation)
	// 	options.loadProvidersFrom(defaultProviderConfigLocation)
	// }

	// Initialize the passive subdomain enumeration engine
	runner.initializeAgent()

	// // Initialize the subdomain resolver
	// err := runner.initializeResolver()
	// if err != nil {
	// 	return nil, err
	// // }

	// Initialize the custom rate limit
	runner.rateLimit = &agent.CustomRateLimit{
		Custom: mapsutil.SyncLockMap[string, uint]{
			Map: make(map[string]uint),
		},
	}

	for source, sourceRateLimit := range options.RateLimits.AsMap() {
		if sourceRateLimit.MaxCount > 0 && sourceRateLimit.MaxCount <= math.MaxUint {
			_ = runner.rateLimit.Custom.Set(source, sourceRateLimit.MaxCount)
		}
	}

	return runner, nil
}

func (r *Runner) initializeAgent() {
	r.agent = agent.New(r.options.Sources, r.options.ExcludeSources, r.options.All)
}

// RunEnumeration wraps RunEnumerationWithCtx with an empty context
func (r *Runner) RunEnumeration() error {
	ctx, _ := contextutil.WithValues(context.Background(), contextutil.ContextArg("All"), contextutil.ContextArg(strconv.FormatBool(r.options.All)))
	return r.RunEnumerationWithCtx(ctx)
}

// RunEnumerationWithCtx runs the domain enumeration flow on the targets specified
func (r *Runner) RunEnumerationWithCtx(ctx context.Context) error {
	outputs := []io.Writer{r.options.Output}

	if len(r.options.Domain) > 0 {
		domainsReader := strings.NewReader(strings.Join(r.options.Domain, "\n"))
		return r.EnumerateMultipleDomainsWithCtx(ctx, domainsReader, outputs)
	}

	// If we have STDIN input, treat it as multiple domains
	if r.options.Stdin {
		return r.EnumerateMultipleDomainsWithCtx(ctx, os.Stdin, outputs)
	}
	return nil
}

// EnumerateMultipleDomains wraps EnumerateMultipleDomainsWithCtx with an empty context
func (r *Runner) EnumerateMultipleDomains(reader io.Reader, writers []io.Writer) error {
	ctx, _ := contextutil.WithValues(context.Background(), contextutil.ContextArg("All"), contextutil.ContextArg(strconv.FormatBool(r.options.All)))
	return r.EnumerateMultipleDomainsWithCtx(ctx, reader, writers)
}

// EnumerateMultipleDomainsWithCtx enumerates subdomains for multiple domains
// We keep enumerating subdomains for a given domain until we reach an error
func (r *Runner) EnumerateMultipleDomainsWithCtx(ctx context.Context, reader io.Reader, writers []io.Writer) error {
	var err error
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		domain := preprocessDomain(scanner.Text())
		domain = replacer.Replace(domain)

		if domain == "" {
			continue
		}

		var file *os.File
		// If the user has specified an output file, use that output file instead
		// of creating a new output file for each domain. Else create a new file
		// for each domain in the directory.
		if r.options.OutputFile != "" {
			outputWriter := NewOutputWriter(r.options.JSON)
			file, err = outputWriter.createFile(r.options.OutputFile, true)
			if err != nil {
				gologger.Error().Msgf("Could not create file %s for %s: %s\n", r.options.OutputFile, r.options.Domain, err)
				return err
			}

			_, err = r.EnumerateSingleDomainWithCtx(ctx, domain, append(writers, file))

			file.Close()
		} else if r.options.OutputDirectory != "" {
			outputFile := path.Join(r.options.OutputDirectory, domain)
			if r.options.JSON {
				outputFile += ".json"
			} else {
				outputFile += ".txt"
			}

			outputWriter := NewOutputWriter(r.options.JSON)
			file, err = outputWriter.createFile(outputFile, false)
			if err != nil {
				gologger.Error().Msgf("Could not create file %s for %s: %s\n", r.options.OutputFile, r.options.Domain, err)
				return err
			}

			_, err = r.EnumerateSingleDomainWithCtx(ctx, domain, append(writers, file))

			file.Close()
		} else {
			_, err = r.EnumerateSingleDomainWithCtx(ctx, domain, writers)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
