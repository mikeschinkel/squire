package gommod

import (
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

var appInfo = appinfo.New(appinfo.Args{
	Name:        gomion.AppName,
	Description: gomion.AppDescr,
	Version:     gomion.Version,
	AppSlug:     gomion.AppSlug,
	ConfigSlug:  gomion.ConfigSlug,
	ConfigFile:  gomion.ConfigFile,
	InfoURL:     gomion.InfoURL,
	ExeName:     gomion.ExeName,
	LogFile:     gomion.LogFile,
	ExtraInfo:   gomion.ExtraInfo,
})

func AppInfo() appinfo.AppInfo {
	return appInfo
}
