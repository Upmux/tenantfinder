package runner

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/hako/durafmt"

	"github.com/projectdiscovery/gologger"

	"github.com/upmux/tenantfinder/pkg/agent"
	"github.com/upmux/tenantfinder/pkg/resolve"
	"github.com/upmux/tenantfinder/pkg/source"
)

const maxNumCount = 2

var replacer = strings.NewReplacer(
	"/", "",
	"•.", "",
	"•", "",
	"*.", "",
	"http://", "",
	"https://", "",
)

// EnumerateSingleDomain wraps EnumerateSingleDomainWithCtx with an empty context
func (r *Runner) EnumerateSingleDomain(domain string, writers []io.Writer) (map[string]map[string]struct{}, error) {
	return r.EnumerateSingleDomainWithCtx(context.Background(), domain, writers)
}

// EnumerateSingleDomainWithCtx performs subdomain enumeration against a single domain
func (r *Runner) EnumerateSingleDomainWithCtx(ctx context.Context, domain string, writers []io.Writer) (map[string]map[string]struct{}, error) {
	gologger.Info().Msgf("Enumerating domains for %s\n", domain)

	now := time.Now()
	results := r.agent.EnumerateDomains(domain, r.options.Proxy, r.options.RateLimit, r.options.Timeout, time.Duration(r.options.MaxEnumerationTime)*time.Minute, agent.WithCustomRateLimit(r.rateLimit))

	wg := &sync.WaitGroup{}
	wg.Add(1)
	// Create a unique map for filtering duplicate domains out
	uniqueMap := make(map[string]resolve.HostEntry)
	// Create a map to track sources for each host
	sourceMap := make(map[string]map[string]struct{})
	skippedCounts := make(map[string]int)
	// Process the results in a separate goroutine
	go func() {
		for result := range results {
			switch result.Type {
			case source.Error:
				gologger.Warning().Msgf("Encountered an error with source %s: %s\n", result.Source, result.Error)
			case source.Domain:
				tenantDomain := replacer.Replace(result.Value)
				tenantDomain = preprocessDomain(tenantDomain)

				if _, ok := uniqueMap[domain]; !ok {
					sourceMap[domain] = make(map[string]struct{})
				}

				// Log the verbose message about the found subdomain per source
				if _, ok := sourceMap[domain][result.Source]; !ok {
					gologger.Verbose().Label(result.Source).Msg(domain)
				}

				sourceMap[domain][result.Source] = struct{}{}

				// Check if the subdomain is a duplicate.
				if _, ok := uniqueMap[domain]; ok {
					skippedCounts[result.Source]++
					continue
				}
				hostEntry := resolve.HostEntry{Domain: domain, Host: tenantDomain, Source: result.Source}
				uniqueMap[tenantDomain] = hostEntry
			}
		}
		// Close the task channel only if wildcards are asked to be removed
		wg.Done()
	}()

	// If the user asked to remove wildcards, listen from the results
	// queue and write to the map. At the end, print the found results to the screen
	// foundResults := make(map[string]resolve.Result)

	wg.Wait()
	outputWriter := NewOutputWriter(r.options.JSON)
	// Now output all results in output writers
	var err error
	for _, writer := range writers {
		if r.options.CaptureSources {
			err = outputWriter.WriteSourceHost(domain, sourceMap, writer)
		} else {
			err = outputWriter.WriteHost(domain, uniqueMap, writer)
		}

		if err != nil {
			gologger.Error().Msgf("Could not write results for %s: %s\n", domain, err)
			return nil, err
		}
	}
	// }

	duration := durafmt.Parse(time.Since(now)).LimitFirstN(maxNumCount).String()
	numberOfSubDomains := len(uniqueMap)

	gologger.Info().Msgf("Found %d domains for %s in %s\n", numberOfSubDomains, domain, duration)

	if r.options.Statistics {
		gologger.Info().Msgf("Printing source statistics for %s", domain)
		statistics := r.agent.GetStatistics()
		// This is a hack to remove the skipped count from the statistics
		// as we don't want to show it in the statistics.
		// TODO: Design a better way to do this.
		for source, count := range skippedCounts {
			if stat, ok := statistics[source]; ok {
				stat.Results -= count
				statistics[source] = stat
			}
		}
		printStatistics(statistics)
	}

	return sourceMap, nil
}
