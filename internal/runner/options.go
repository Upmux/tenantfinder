package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/upmux/tenantfinder/pkg/agent"
	"github.com/upmux/tenantfinder/pkg/resolve"

	"github.com/projectdiscovery/goflags"
	"github.com/projectdiscovery/gologger"
	fileutil "github.com/projectdiscovery/utils/file"
	folderutil "github.com/projectdiscovery/utils/folder"
	logutil "github.com/projectdiscovery/utils/log"
)

var (
	configDir                     = folderutil.AppConfigDirOrDefault(".", "tenantfinder")
	defaultConfigLocation         = filepath.Join(configDir, "config.yaml")
	defaultProviderConfigLocation = filepath.Join(configDir, "provider-config.yaml")
)

// Options contains the configuration options for tuning
// the domain enumeration process.
type Options struct {
	Verbose            bool                // Verbose flag indicates whether to show verbose output or not
	NoColor            bool                // NoColor disables the colored output
	JSON               bool                // JSON specifies whether to use json for output format or text file
	Silent             bool                // Silent suppresses any extra text and only writes domains to screen
	ListSources        bool                // ListSources specifies whether to list all available sources
	CaptureSources     bool                // CaptureSources specifies whether to save all sources that returned a specific domains or just the first source
	Stdin              bool                // Stdin specifies whether stdin input was given to the process
	Version            bool                // Version specifies if we should just show version and exit
	All                bool                // All specifies whether to use all (slow) sources.
	Statistics         bool                // Statistics specifies whether to report source statistics
	Timeout            int                 // Timeout is the seconds to wait for sources to respond
	MaxEnumerationTime int                 // MaxEnumerationTime is the maximum amount of time in minutes to wait for enumeration
	Domain             goflags.StringSlice // Domain is the domain to find subdomains for
	Output             io.Writer
	OutputFile         string               // Output is the file to write found domains to.
	OutputDirectory    string               // OutputDirectory is the directory to write results to in case list of domains is given
	Sources            goflags.StringSlice  `yaml:"sources,omitempty"`         // Sources contains a comma-separated list of sources to use for enumeration
	ExcludeSources     goflags.StringSlice  `yaml:"exclude-sources,omitempty"` // ExcludeSources contains the comma-separated sources to not include in the enumeration process
	Config             string               // Config contains the location of the config file
	ProviderConfig     string               // ProviderConfig contains the location of the provider config file
	Proxy              string               // HTTP proxy
	RateLimit          int                  // Global maximum number of HTTP requests to send per second
	RateLimits         goflags.RateLimitMap // Maximum number of HTTP requests to send per second
	ResultCallback     OnResultCallback     // OnResult callback
}

// OnResultCallback (hostResult)
type OnResultCallback func(result *resolve.HostEntry)

// ParseOptions parses the command line flags provided by a user
func ParseOptions() *Options {
	logutil.DisableDefaultLogger()

	options := &Options{}

	var err error
	flagSet := goflags.NewFlagSet()
	flagSet.SetDescription(`A streamlined tool for discovering related domains.`)

	flagSet.CreateGroup("input", "Input",
		flagSet.StringSliceVarP(&options.Domain, "domain", "d", nil, "domains to find subdomains for", goflags.NormalizedStringSliceOptions),
	)

	flagSet.CreateGroup("source", "Source",
		flagSet.StringSliceVarP(&options.Sources, "sources", "s", nil, "specific sources to use for discovery (-s aad). Use -ls to display all available sources.", goflags.NormalizedStringSliceOptions),
		flagSet.StringSliceVarP(&options.ExcludeSources, "exclude-sources", "es", nil, "sources to exclude from enumeration (-es aad)", goflags.NormalizedStringSliceOptions),
		flagSet.BoolVar(&options.All, "all", false, "use all sources for enumeration (slow)"),
	)

	flagSet.CreateGroup("rate-limit", "Rate-limit",
		flagSet.IntVarP(&options.RateLimit, "rate-limit", "rl", 0, "maximum number of http requests to send per second (global)"),
		flagSet.RateLimitMapVarP(&options.RateLimits, "rate-limits", "rls", defaultRateLimits, "maximum number of http requests to send per second four providers in key=value format (-rls aad=10/m)", goflags.NormalizedStringSliceOptions),
	)

	flagSet.CreateGroup("output", "Output",
		flagSet.StringVarP(&options.OutputFile, "output", "o", "", "file to write output to"),
		flagSet.BoolVarP(&options.JSON, "jsonl", "j", false, "write output in JSONL(ines) format"),
		flagSet.StringVarP(&options.OutputDirectory, "output-dir", "od", "", "directory to write output file"),
		flagSet.BoolVarP(&options.CaptureSources, "collect-sources", "cs", false, "include all sources in the output (-json only)"),
	)

	flagSet.CreateGroup("configuration", "Configuration",
		// flagSet.StringVar(&options.Config, "config", defaultConfigLocation, "flag config file"),
		// flagSet.StringVarP(&options.ProviderConfig, "provider-config", "pc", defaultProviderConfigLocation, "provider config file"),
		flagSet.StringVar(&options.Proxy, "proxy", "", "http proxy to use with tenantfinder"),
	)

	flagSet.CreateGroup("debug", "Debug",
		flagSet.BoolVar(&options.Silent, "silent", false, "show only domains in output"),
		flagSet.BoolVar(&options.Version, "version", false, "show version of tenantfinder"),
		flagSet.BoolVar(&options.Verbose, "v", false, "show verbose output"),
		flagSet.BoolVarP(&options.NoColor, "no-color", "nc", false, "disable color in output"),
		flagSet.BoolVarP(&options.ListSources, "list-sources", "ls", false, "list all available sources"),
		flagSet.BoolVar(&options.Statistics, "stats", false, "report source statistics"),
	)

	flagSet.CreateGroup("optimization", "Optimization",
		flagSet.IntVar(&options.Timeout, "timeout", 30, "seconds to wait before timing out"),
		flagSet.IntVar(&options.MaxEnumerationTime, "max-time", 10, "minutes to wait for enumeration results"),
	)

	if err := flagSet.Parse(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// if exists := fileutil.FileExists(defaultProviderConfigLocation); !exists {
	// 	if err := createProviderConfigYAML(defaultProviderConfigLocation); err != nil {
	// 		gologger.Error().Msgf("Could not create provider config file: %s\n", err)
	// 	}
	// }

	// if options.Config != defaultConfigLocation {
	// 	// An empty source file is not a fatal error
	// 	if err := flagSet.MergeConfigFile(options.Config); err != nil && !errors.Is(err, io.EOF) {
	// 		gologger.Fatal().Msgf("Could not read config: %s\n", err)
	// 	}
	// }

	// Default output is stdout
	options.Output = os.Stdout

	// Check if stdin pipe was given
	options.Stdin = fileutil.HasStdin()

	// Read the inputs and configure the logging
	options.configureOutput()

	if options.Version {
		gologger.Info().Msgf("Current Version: %s\n", version)
		gologger.Info().Msgf("tenantfinder Config Directory: %s", configDir)
		os.Exit(0)
	}

	options.preProcessDomains()

	if !options.Silent {
		showBanner()
	}

	if options.ListSources {
		listSources(options)
		os.Exit(0)
	}

	// Validate the options passed by the user and if any
	// invalid options have been used, exit.
	err = options.validateOptions()
	if err != nil {
		gologger.Warning().Msgf("Program exiting: %s\n", err)
		os.Exit(0)
	}

	return options
}

// loadProvidersFrom runs the app with source config
func (options *Options) loadProvidersFrom(location string) {

	// We skip bailing out if file doesn't exist because we'll create it
	// at the end of options parsing from default via goflags.
	if err := UnmarshalFrom(location); err != nil && (!strings.Contains(err.Error(), "file doesn't exist") || errors.Is(err, os.ErrNotExist)) {
		gologger.Error().Msgf("Could not read providers from %s: %s\n", location, err)
	}
}

func listSources(options *Options) {
	gologger.Info().Msgf("Current list of available sources. [%d]\n", len(agent.AllSources))
	gologger.Info().Msgf("Sources marked with an * need key(s) or token(s) to work.\n")
	gologger.Info().Msgf("You can modify %s to configure your keys/tokens.\n\n", options.ProviderConfig)

	for _, source := range agent.AllSources {
		message := "%s\n"
		sourceName := source.Name()
		if source.NeedsKey() {
			message = "%s *\n"
		}
		gologger.Silent().Msgf(message, sourceName)
	}
}

func (options *Options) preProcessDomains() {
	for i, domain := range options.Domain {
		options.Domain[i] = preprocessDomain(domain)
	}
}

var defaultRateLimits = []string{
	"aad=10/m",
}
