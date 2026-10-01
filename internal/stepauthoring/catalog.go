package stepauthoring

import (
	"fmt"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/apitools/sqlitecache"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

// CatalogInstallation is trusted CLI configuration, never a model/request field.
// APItools owns root validation, registrations, index identity and ranking.
type CatalogInstallation struct {
	Root          catalog.RootOptions
	MetadataPath  string
	RemoteEnabled bool
}

func (c CatalogInstallation) IndexOptions() (apitools.CatalogIndexOptions, error) {
	options := apitools.CatalogIndexOptions{Root: c.Root, ReadRegistrations: sqlitecache.ReadCatalogSpecArtifacts}
	if c.MetadataPath == "" {
		return options, nil
	}
	data, _, err := evidencefile.ReadRegular(c.MetadataPath, 2<<20)
	if err != nil {
		return options, fmt.Errorf("catalog installation metadata is unavailable")
	}
	var value catalog.Catalog
	if evidencefile.DecodeStrict(data, &value) != nil || value.Validate() != nil {
		return options, fmt.Errorf("catalog installation metadata is invalid")
	}
	options.Catalog = &value
	return options, nil
}
