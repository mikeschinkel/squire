package goutils

type Require struct {
	PathVersion
	Indirect bool
}

func NewRequire(pv PathVersion, indirect bool) Require {
	return Require{PathVersion: pv, Indirect: indirect}
}
