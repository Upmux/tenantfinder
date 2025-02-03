package runner

import (
	"github.com/projectdiscovery/gologger"
)

const banner = `
  ______                       __  _______           __         
 /_  __/__  ____  ____ _____  / /_/ ____(_)___  ____/ /__  _____
  / / / _ \/ __ \/ __ \/ __ \/ __/ /_  / / __ \/ __  / _ \/ ___/
 / / /  __/ / / / /_/ / / / / /_/ __/ / / / / / /_/ /  __/ /    
/_/  \___/_/ /_/\__,_/_/ /_/\__/_/   /_/_/ /_/\__,_/\___/_/     
`

// Name
const ToolName = `tenantfinder`

// Version is the current version of tenantfinder
const version = `v0.0.1`

// showBanner is used to show the banner to the user
func showBanner() {
	gologger.Print().Msgf("%s\n", banner)
	gologger.Print().Msgf("\t\t\tupmux.com\n\n")
}

func GetUpdateCallback() func() {
	return func() {
		showBanner()
	}
}
