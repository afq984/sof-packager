package internal

import (
	"github.com/protocolbuffers/txtpbfmt/parser"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func textproto(m proto.Message) ([]byte, error) {
	b, err := prototext.Marshal(m)
	if err != nil {
		return nil, err
	}
	b, err = parser.FormatWithConfig(b, parser.Config{ExpandAllChildren: true})
	if err != nil {
		return nil, err
	}
	return b, err
}
