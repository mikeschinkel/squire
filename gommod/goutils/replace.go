package goutils

type Replace struct {
	Old PathVersion
	New PathVersion
}

func NewReplace(old, new PathVersion) Replace {
	return Replace{
		Old: old,
		New: new,
	}
}
