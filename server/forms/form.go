/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

// Package forms contains methods for handling form decoding and encoding.
package forms

import (
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"

	"github.com/go-playground/form/v4"
	"github.com/immanent-tech/go-base/validation"
)

var (
	// ErrDecode indicates an error occurred during decoding.
	ErrDecode = errors.New("error in decoding")
	// ErrEncode indicates an error occurred during encoding.
	ErrEncode = errors.New("error in encoding")
	// ErrValidation indicates an error occurred during validation.
	ErrValidation = errors.New("validation failed")
	// ErrNoFormData indicates that no form data was parsed.
	ErrNoFormData = errors.New("no form data")
	// ErrSanitise indicates an error occurred during sanitisation.
	ErrSanitise = errors.New("sanitisation failed")
	// ErrFileTooLarge can be used to indicate a file upload had a size greater than a set limit.
	ErrFileTooLarge = errors.New("file is too large")
)

var (
	decoder = form.NewDecoder()
)

// defaultMaxMemory for a multipart for submission is 32 MB.
const defaultMaxMemory = 32 << 20

// FormInput represents form input data. It has methods to test if the data is valid and to sanitise the input data.
type FormInput interface {
	Validate() error
	Sanitise() error
}

// DecodeForm will decode submitted form contents into the passed in type. It
// will perform validation of the type and will return the type and a boolean
// true if it is valid. If decoding the form submission fails, a non-nill error
// is returned.
func DecodeForm[T FormInput](req *http.Request) (T, error) {
	if err := req.ParseForm(); err != nil {
		var obj T
		return obj, fmt.Errorf("%w: %w", ErrDecode, err)
	}
	switch req.Method {
	case http.MethodGet:
		if len(req.Form) == 0 {
			var obj T
			return obj, nil
		}
	case http.MethodPost:
		if len(req.PostForm) == 0 {
			var obj T
			return obj, nil
		}
	}
	obj, err := decodeObject[T](req)
	if err != nil {
		return obj, fmt.Errorf("%w: %w", ErrDecode, err)
	}
	return obj, nil
}

func DecodeMultiPartForm[T FormInput](req *http.Request) (T, error) {
	if err := req.ParseMultipartForm(defaultMaxMemory); err != nil {
		var obj T
		return obj, fmt.Errorf("%w: %w", ErrDecode, err)
	}
	obj, err := decodeObject[T](req)
	if err != nil {
		return obj, fmt.Errorf("%w: %w", ErrDecode, err)
	}
	return obj, nil
}

func decodeObject[T FormInput](req *http.Request) (T, error) {
	var obj T
	// Decode the form values.
	if err := decoder.Decode(&obj, req.Form); err != nil {
		return obj, fmt.Errorf("%w: %w", ErrDecode, err)
	}
	// Sanitise the object.
	if err := obj.Sanitise(); err != nil {
		return obj, fmt.Errorf("%w: %w", ErrSanitise, err)
	}
	// Validate the object.
	if err := obj.Validate(); err != nil {
		return obj, fmt.Errorf("%w: %w", ErrValidation, err)
	}
	return obj, nil
}

// DecodeMultipartValue will the file represented by the given field in a multipart form submission. It will perform
// validation of the file and will return the file object and a boolean true if it is valid. If decoding fails, a
// non-nil error is returned.
func DecodeMultipartValue(req *http.Request, field string) (string, error) {
	// Parse form values in request.
	if err := req.ParseMultipartForm(defaultMaxMemory); err != nil {
		return "", errors.Join(ErrDecode, err)
	}
	// Decode the form values.
	return validation.SanitizeInput(req.FormValue(field)), nil
}

// DecodeMultipartFile will decode the file represented by the given field in a multipart form submission. It will
// return the data as a [FileUpload]. If decoding fails, a non-nil error is returned.
func DecodeMultipartFile(req *http.Request, field string) (*FileUpload, error) {
	// Parse form values in request.
	if err := req.ParseMultipartForm(defaultMaxMemory); err != nil {
		return nil, fmt.Errorf("decode multipart form: %w", err)
	}
	// Decode the form values.
	data, hdr, err := req.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("decode form file: %w", err)
	}
	// Create a FileUpload object.
	upload := &FileUpload{
		File:       data,
		FileHeader: hdr,
	}
	// Validate file upload.
	if err := upload.Validate(); err != nil {
		return nil, fmt.Errorf("validate file upload: %w", err)
	}
	return upload, nil
}

type FileUpload struct {
	*multipart.FileHeader `json:"header" validate:"required"`
	multipart.File        `json:"data"   validate:"required"`
}

func (f FileUpload) Validate() error {
	if err := validation.Validate.Struct(f); err != nil {
		return fmt.Errorf("invalid file upload: %w", err)
	}
	return nil
}

// GetSize returns the file size.
func (f FileUpload) GetSize() int64 {
	if f.FileHeader != nil {
		return f.FileHeader.Size
	}
	return 0
}

func (f FileUpload) GetFilename() string {
	return f.Filename
}

func (f FileUpload) GetHeader(key string) string {
	return f.FileHeader.Header.Get(key)
}

// ParseMimetype attempts to parse and return the mime type of the file from its mime header.
func (f FileUpload) ParseMimetype() (string, error) {
	if f.FileHeader == nil {
		return "unknown", errors.New("cannot parse mime type of file: invalid file")
	}
	mediaType, _, err := mime.ParseMediaType(f.GetHeader("Content-Type"))
	if err != nil {
		return "unknown", fmt.Errorf("cannot parse mime type of file %s: %w", f.GetFilename(), err)
	}
	return mediaType, nil
}
