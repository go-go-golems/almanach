// Package layoutpb is the Go side of the Almanach Layout DSL v2 contract. It
// wraps the protobuf-generated layout types (gen/almanach/layout/v1) with a
// small JSON codec whose options match what the TypeScript studio expects:
// camelCase keys via protojson (UseProtoNames:false), so @bufbuild/protobuf
// fromJson decodes the same bytes.
package layoutpb

import (
	"fmt"

	layoutv1 "github.com/go-go-golems/almanach/gen/almanach/layout/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// SchemaVersionV1 is the current layout schema version. A Layout with an unset
// schema_version is normalized to this value.
const SchemaVersionV1 = 1

// Normalize fills in defaults consumers rely on (currently just schema_version).
func Normalize(layout *layoutv1.Layout) (*layoutv1.Layout, error) {
	if layout == nil {
		return nil, fmt.Errorf("layout is nil")
	}
	if layout.SchemaVersion == 0 {
		layout.SchemaVersion = SchemaVersionV1
	}
	return layout, nil
}

// MarshalJSON encodes a Layout to the wire JSON the studio consumes. Keys are
// camelCase (UseProtoNames:false); unset fields are omitted.
func MarshalJSON(layout *layoutv1.Layout) ([]byte, error) {
	layout, err := Normalize(layout)
	if err != nil {
		return nil, err
	}
	return protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: false,
	}.Marshal(layout)
}

// UnmarshalJSON decodes wire JSON into a Layout. Unknown fields are rejected so
// schema drift surfaces instead of vanishing silently.
func UnmarshalJSON(data []byte) (*layoutv1.Layout, error) {
	layout := &layoutv1.Layout{}
	if err := (protojson.UnmarshalOptions{
		DiscardUnknown: false,
	}).Unmarshal(data, layout); err != nil {
		return nil, err
	}
	return Normalize(layout)
}

// MarshalBinary / UnmarshalBinary give a compact wire form for internal use.
func MarshalBinary(layout *layoutv1.Layout) ([]byte, error) {
	layout, err := Normalize(layout)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(layout)
}

func UnmarshalBinary(data []byte) (*layoutv1.Layout, error) {
	layout := &layoutv1.Layout{}
	if err := proto.Unmarshal(data, layout); err != nil {
		return nil, err
	}
	return Normalize(layout)
}
