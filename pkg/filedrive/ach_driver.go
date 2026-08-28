package filedrive

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/moov-io/ach"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"goftp.io/server/v2"
)

// ACHDriver wraps the goftp driver to add additional logic and error checking.
type ACHDriver struct {
	server.Driver

	logger       log.Logger
	validateOpts *ach.ValidateOpts
}

func NewACHDriver(logger log.Logger, validateOpts *ach.ValidateOpts, driver server.Driver) *ACHDriver {
	return &ACHDriver{
		Driver:       driver,
		logger:       logger,
		validateOpts: validateOpts,
	}
}

// // PutFile overrides the existing method to prevent erroneous ACH files from being uploaded.
// // offset follows goftp.io/server/v2 semantics: -1 overwrites, >= 0 appends/seeks.
// func (d *ACHDriver) PutFile(ctx *server.Context, path string, r io.Reader, offset int64) (int64, error) {
// 	_, span := telemetry.StartSpan(context.Background(), "put-file", trace.WithAttributes(
// 		attribute.String("ftp.destination", path),
// 	))
// 	defer span.End()

// 	d.logger.Info().Log(fmt.Sprintf("receiving file for %s", path))

// 	// // Read the file that was uploaded
// 	// var buf bytes.Buffer
// 	// tee := io.TeeReader(r, &buf)

// 	// reader := ach.NewReader(tee)
// 	// reader.SetValidation(d.validateOpts)

// 	// file, err := reader.Read()
// 	// if err != nil {
// 	// 	span.RecordError(err)
// 	// 	d.logger.Error().Log(fmt.Sprintf("ftp: error reading ACH file %s: %v", path, err))
// 	// 	return 0, err
// 	// }

// 	var buf bytes.Buffer

// 	d.logger.Info().Log(fmt.Sprintf(
// 		"FTP PutFile: incoming reader type=%T",
// 		r,
// 	))

// 	tee := io.TeeReader(r, &buf)

// 	d.logger.Info().Log("FTP PutFile: creating ACH reader")

// 	reader := ach.NewReader(tee)

// 	d.logger.Info().Log("FTP PutFile: ACH reader created")

// 	reader.SetValidation(d.validateOpts)

// 	file, err := reader.Read()

// 	d.logger.Info().Log(fmt.Sprintf(
// 		"FTP PutFile: ACH reader finished, buffer size=%d, err=%v",
// 		buf.Len(),
// 		err,
// 	))

// 	if err != nil {
// 		span.RecordError(err)
// 		d.logger.Error().Log(fmt.Sprintf(
// 			"ftp: error reading ACH file %s: %v",
// 			path,
// 			err,
// 		))
// 		return 0, err
// 	}

// 	if err := file.Create(); err != nil {
// 		d.logger.Error().Log(fmt.Sprintf("ftp: error creating file %s: %v", path, err))
// 		return 0, err
// 	}

// 	span.SetAttributes(attribute.Int("ftp.file_size_bytes", buf.Len()))
// 	d.logger.Info().Log(fmt.Sprintf("accepting file at %s", path))

// 	// Call the original PutFile method with a reset reader.
// 	return d.Driver.PutFile(ctx, path, &buf, offset)
// }

// PutFile overrides the existing method to prevent erroneous ACH files from being uploaded.
// offset follows goftp.io/server/v2 semantics: -1 overwrites, >= 0 appends/seeks.
func (d *ACHDriver) PutFile(ctx *server.Context, path string, r io.Reader, offset int64) (int64, error) {
	_, span := telemetry.StartSpan(context.Background(), "put-file", trace.WithAttributes(
		attribute.String("ftp.destination", path),
	))
	defer span.End()

	d.logger.Info().Log(fmt.Sprintf("receiving file for %s", path))

	d.logger.Info().Log(fmt.Sprintf(
		"FTP PutFile: incoming reader type=%T",
		r,
	))

	// Diagnostic: read directly from the FTP data connection.
	raw, err := io.ReadAll(r)

	d.logger.Info().Log(fmt.Sprintf(
		"FTP PutFile: io.ReadAll finished, bytes=%d, err=%v, content=%q",
		len(raw),
		err,
		raw,
	))

	if err != nil {
		span.RecordError(err)
		d.logger.Error().Log(fmt.Sprintf(
			"ftp: error reading FTP data for %s: %v",
			path,
			err,
		))
		return 0, err
	}

	// Parse the bytes that were received from FTP.
	var buf bytes.Buffer
	buf.Write(raw)

	d.logger.Info().Log("FTP PutFile: creating ACH reader")

	reader := ach.NewReader(bytes.NewReader(raw))

	d.logger.Info().Log("FTP PutFile: ACH reader created")

	reader.SetValidation(d.validateOpts)

	file, err := reader.Read()

	d.logger.Info().Log(fmt.Sprintf(
		"FTP PutFile: ACH reader finished, buffer size=%d, err=%v",
		buf.Len(),
		err,
	))

	if err != nil {
		span.RecordError(err)
		d.logger.Error().Log(fmt.Sprintf(
			"ftp: error reading ACH file %s: %v",
			path,
			err,
		))
		return 0, err
	}

	if err := file.Create(); err != nil {
		d.logger.Error().Log(fmt.Sprintf(
			"ftp: error creating file %s: %v",
			path,
			err,
		))
		return 0, err
	}

	span.SetAttributes(attribute.Int("ftp.file_size_bytes", buf.Len()))

	d.logger.Info().Log(fmt.Sprintf(
		"accepting file at %s",
		path,
	))

	// Call the original PutFile method with the received bytes.
	return d.Driver.PutFile(ctx, path, bytes.NewReader(raw), offset)
}

// Ensure ACHDriver implements server.Driver (compile-time check).
var _ server.Driver = (*ACHDriver)(nil)
