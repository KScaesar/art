package main

const MetadataTmpl = `
package {{.Package}}

import (
	"github.com/KScaesar/art"
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

func (key *{{.FileName}}MetadataKey) GetCorrelationId(message *art.Message) string {
	md := message.Metadata
	return md.Get(key.corId).(string)
}

func (key *{{.FileName}}MetadataKey) SetCorrelationId(message *art.Message, value string) {
	md := message.Metadata
	md.Set(key.corId, value)
}
`
