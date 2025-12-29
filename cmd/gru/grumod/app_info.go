package grumod

import (
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/squire/gru/grumod/gru"
)

var appInfo = appinfo.New(appinfo.Args{
	Name:        gru.AppName,
	Description: gru.AppDescr,
	Version:     gru.Version,
	AppSlug:     gru.AppSlug,
	ConfigSlug:  gru.ConfigSlug,
	ConfigFile:  gru.ConfigFile,
	InfoURL:     gru.InfoURL,
	ExeName:     gru.ExeName,
	LogFile:     gru.LogFile,
	LogPath:     gru.LogPath,
	ExtraInfo:   gru.ExtraInfo,
})

func AppInfo() appinfo.AppInfo {
	return appInfo
}
