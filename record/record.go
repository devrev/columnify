package record

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/devrev/columnify/schema"
)

const (
	RecordTypeAvro    = "avro"
	RecordTypeCsv     = "csv"
	RecordTypeJsonl   = "jsonl"
	RecordTypeLtsv    = "ltsv"
	RecordTypeMsgpack = "msgpack"
	RecordTypeTsv     = "tsv"
)

var (
	ErrUnsupportedRecord   = errors.New("unsupported record")
	ErrUnconvertibleRecord = errors.New("input record is unable to convert")
)

// innerDecoder decodes data from given Reader to the intermediate representation.
type innerDecoder interface {
	// Decode reads input data via Reader and extract it to the argument.
	// If there is no data left to be read, Read returns nil, io.EOF.
	Decode(r *map[string]interface{}) error
}

// jsonStringConverter converts data with innerDecoder and returns JSON string value.
type jsonStringConverter struct {
	inner innerDecoder
}

func NewJsonStringConverter(r io.Reader, s *schema.IntermediateSchema, recordType string) (*jsonStringConverter, error) {
	var inner innerDecoder
	var err error

	switch recordType {
	case RecordTypeAvro:
		inner, err = newAvroInnerDecoder(r)

	case RecordTypeCsv:
		inner, err = newCsvInnerDecoder(r, s, CsvDelimiter)

	case RecordTypeJsonl:
		inner = newJsonlInnerDecoder(r)

	case RecordTypeLtsv:
		inner = newLtsvInnerDecoder(r)

	case RecordTypeMsgpack:
		inner = newMsgpackInnerDecoder(r)

	case RecordTypeTsv:
		inner, err = newCsvInnerDecoder(r, s, TsvDelimiter)

	default:
		return nil, fmt.Errorf("unsupported record type %s: %w", recordType, ErrUnsupportedRecord)
	}

	return &jsonStringConverter{
		inner: inner,
	}, err
}

func (d *jsonStringConverter) Convert(v *string) error {
	var vv map[string]interface{}

	err := d.inner.Decode(&vv)
	if err != nil {
		return err
	}

	data, err := json.Marshal(vv)
	if err != nil {
		return err
	}
	*v = string(data)

	return nil
}
