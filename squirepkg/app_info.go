package squirepkg

import (
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/squire/squirepkg/squire"
)

var appInfo = appinfo.New(appinfo.Args{
	Name:        squire.AppName,
	Description: squire.AppDescr,
	Version:     squire.Version,
	AppSlug:     squire.AppSlug,
	ConfigSlug:  squire.ConfigSlug,
	ConfigFile:  squire.ConfigFile,
	InfoURL:     squire.InfoURL,
	ExeName:     squire.ExeName,
	LogFile:     squire.LogFile,
	ExtraInfo:   squire.ExtraInfo,
})

func AppInfo() appinfo.AppInfo {
	return appInfo
}
