package squirecfg

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/squire/squirepkg/common"
)

const (
	RootConfigV1Version = 1
	RootConfigV1Schema  = "https://squire.github.io/schemas/v1/root-schema.json"
)

var _ Config = (*RootConfigV1)(nil)
var _ cfgstore.RootConfig = (*RootConfigV1)(nil)

// RootConfigV1 represents the root configuration structure as defined in ADR-001
type RootConfigV1 struct {
	rootConfigV1Base `json:",inline"`
}

func (c *RootConfigV1) Merge(rc cfgstore.RootConfig) cfgstore.RootConfig {
	// Nothing to do, yet. Note that this is returning `rc` instead of `c` which
	// means it is return CLI Config and not Project Config
	return rc
}

func (c *RootConfigV1) RootConfig() {}

// Base struct with non-polymorphic fields
type rootConfigV1Base struct {
	Schema  string `json:"$schema"`
	Version int    `json:"version"`
}

//goland:noinspection GoUnusedExportedFunction
func NewRootConfigV1() (c *RootConfigV1) {
	return &RootConfigV1{
		rootConfigV1Base: rootConfigV1Base{
			Schema:  RootConfigV1Schema,
			Version: RootConfigV1Version,
		},
	}
}

func (c *RootConfigV1) Config() {}

func (c *RootConfigV1) Normalize(cfgstore.NormalizeArgs) (err error) {
	c.Schema = RootConfigV1Schema
	c.Version = RootConfigV1Version
	return err
}

func (c *RootConfigV1) String() string {
	return string(c.Bytes())
}

func (c *RootConfigV1) Bytes() []byte {
	b, err := jsonv2.Marshal(c, jsontext.WithIndent("  "))
	if err != nil {
		panic(err)
	}
	return b
}

type LoadRootConfigV1Args struct {
	AppInfo appinfo.AppInfo
	Options cfgstore.Options
}

func LoadRootConfigV1(args LoadRootConfigV1Args) (_ *RootConfigV1, err error) {
	var dirTypes = []cfgstore.DirType{cfgstore.CLIConfigDirType}

	configStores := cfgstore.NewConfigStores(cfgstore.ConfigStoresArgs{
		DirTypes: dirTypes,
		ConfigStoreArgs: cfgstore.ConfigStoreArgs{
			ConfigSlug:  common.ConfigSlug,
			RelFilepath: common.ConfigFile,
		},
	})

	// Get externally set options such as via the switches on the command line
	return cfgstore.LoadRootConfig[RootConfigV1, *RootConfigV1](configStores, cfgstore.RootConfigArgs{
		DirTypes: dirTypes,
		Options:  args.Options,
	})

}
