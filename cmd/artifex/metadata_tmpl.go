package main

const MetadataTmpl = `
package {{.Package}}

import (
	"github.com/gookit/goutil/maputil"
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

func (key *{{.FileName}}MetadataKey) PingPong(md maputil.Data) (pingpong chan error) {
	return md.Get(key.pingpong).(chan error)
}

func (key *{{.FileName}}MetadataKey) SetPingPong(md maputil.Data, value chan error) {
	md.Set(key.pingpong, value)
}
`
