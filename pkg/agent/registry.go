package agent

import (
	"github.com/upmux/tenantfinder/pkg/source"
	"github.com/upmux/tenantfinder/pkg/source/aad"

	mapsutil "github.com/projectdiscovery/utils/maps"
)

var AllSources = map[string]source.Source{
	"aad": &aad.Source{},
}

var sourceWarnings = mapsutil.NewSyncLockMap[string, string](
	mapsutil.WithMap(mapsutil.Map[string, string]{}))
