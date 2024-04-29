package main

const MetadataTmpl = `
package {{.Package}}

import (
	"github.com/gookit/goutil/maputil"

	"github.com/KScaesar/Artifex"
)

var {{.FileName}}Metadata = new{{.FileName}}MetadataKey()

func new{{.FileName}}MetadataKey() *{{.FileName}}MetadataKey {
	return &{{.FileName}}MetadataKey{
		pingpong: "pingpong",
	}
}

type {{.FileName}}MetadataKey struct {
	pingpong string
}

func (key *{{.FileName}}MetadataKey) PingPong(md maputil.Data) (pingpong Artifex.WaitPingPong) {
	return md.Get(key.pingpong).(Artifex.WaitPingPong)
}

func (key *{{.FileName}}MetadataKey) SetPingPong(md maputil.Data, value Artifex.WaitPingPong) {
	md.Set(key.pingpong, value)
}
`
