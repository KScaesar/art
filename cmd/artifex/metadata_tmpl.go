package main

const MetadataTmpl = `
package {{.Package}}

import (
	"github.com/gookit/goutil/maputil"
)

var {{.FileName}}Metadata = new{{.FileName}}MetadataKey()

func new{{.FileName}}MetadataKey() *{{.FileName}}MetadataKey {
	return &{{.FileName}}MetadataKey{
		corId: "corId",
	}
}

type {{.FileName}}MetadataKey struct {
	corId string
}

func (key *{{.FileName}}MetadataKey) GetCorrelationId(md maputil.Data) any {
	return md.Get(key.corId).(any)
}

func (key *{{.FileName}}MetadataKey) SetCorrelationId(md maputil.Data, value any) {
	md.Set(key.corId, value)
}
`
