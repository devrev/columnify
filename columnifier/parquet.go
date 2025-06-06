package columnifier

import (
	"io"
	"os"

	"github.com/xitongsys/parquet-go/marshal"

	"github.com/devrev/columnify/record"

	"github.com/devrev/columnify/parquet"
	"github.com/devrev/columnify/schema"
	"github.com/xitongsys/parquet-go-source/local"
	parquetSource "github.com/xitongsys/parquet-go/source"
	"github.com/xitongsys/parquet-go/writer"
)

// Columnifier is a parquet specific Columninifier implementation.
type parquetColumnifier struct {
	w      *writer.ParquetWriter
	schema *schema.IntermediateSchema
	rt     string
}

// NewParquetColumnifier creates a new parquetColumnifier.
func NewParquetColumnifier(st string, sf string, rt string, output string, config Config) (*parquetColumnifier, error) {
	schemaContent, err := os.ReadFile(sf)
	if err != nil {
		return nil, err
	}

	intermediateSchema, err := schema.GetSchema(schemaContent, st)
	if err != nil {
		return nil, err
	}

	sh, err := schema.NewSchemaHandlerFromArrow(*intermediateSchema)
	if err != nil {
		return nil, err
	}

	var fw parquetSource.ParquetFile
	if output != "" {
		fw, err = local.NewLocalFileWriter(output)
		if err != nil {
			return nil, err
		}
	} else {
		fw = parquet.NewStdioFile()
	}

	w, err := writer.NewParquetWriter(fw, nil, 1)
	if err != nil {
		return nil, err
	}
	w.SchemaHandler = sh
	w.Footer.Schema = append(w.Footer.Schema, sh.SchemaElements...)

	w.PageSize = config.Parquet.PageSize
	w.RowGroupSize = config.Parquet.RowGroupSize
	w.CompressionType = config.Parquet.CompressionCodec

	// Intermediate record type is string typed JSON values
	w.MarshalFunc = marshal.MarshalJSON

	return &parquetColumnifier{
		w:      w,
		schema: intermediateSchema,
		rt:     rt,
	}, nil
}

// Write reads, converts input binary data and write it to buffer.
func (c *parquetColumnifier) WriteFromReader(reader io.Reader) (int, error) {
	decoder, err := record.NewJsonStringConverter(reader, c.schema, c.rt)
	if err != nil {
		return -1, err
	}

	beforeSize := c.w.Size
	for {
		var v string
		err = decoder.Convert(&v)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return -1, err
			}
		}

		if err := c.w.Write(v); err != nil {
			return -1, err
		}
	}
	afterSize := c.w.Size

	return int(afterSize - beforeSize), nil
}

// writeFromFile reads, converts an input binary file.
func (c *parquetColumnifier) writeFromFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return -1, err
	}
	defer f.Close()

	n, err := c.WriteFromReader(f)
	if err != nil {
		return -1, err
	}

	return n, nil
}

// WriteFromFiles reads, converts input binary files.
func (c *parquetColumnifier) WriteFromFiles(paths []string) (int, error) {
	var size int
	for _, p := range paths {
		n, err := c.writeFromFile(p)
		if err != nil {
			return -1, err
		}
		size += n
	}
	return size, nil
}

// Close stops writing parquet files ant finalize this conversion.
func (c *parquetColumnifier) Close() error {
	if err := c.w.WriteStop(); err != nil {
		return err
	}

	return c.w.PFile.Close()
}
