package service

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrNotFound        = errors.New("resource not found")
	ErrReportTooLarge  = errors.New("sales report is too large")
)
