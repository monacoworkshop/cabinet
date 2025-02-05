package errors

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/golang/protobuf/ptypes/any"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Code ...
type Code int

const (
	// grpc error codes
	NoError            = Code(codes.OK)
	Canceled           = Code(codes.Canceled)
	Unknown            = Code(codes.Unknown)
	InvalidArgument    = Code(codes.InvalidArgument)
	DeadlineExceeded   = Code(codes.DeadlineExceeded)
	NotFound           = Code(codes.NotFound)
	AlreadyExists      = Code(codes.AlreadyExists)
	PermissionDenied   = Code(codes.PermissionDenied)
	ResourceExhausted  = Code(codes.ResourceExhausted)
	FailedPrecondition = Code(codes.FailedPrecondition)
	Aborted            = Code(codes.Aborted)
	OutOfRange         = Code(codes.OutOfRange)
	Unimplemented      = Code(codes.Unimplemented)
	Internal           = Code(codes.Internal)
	Unavailable        = Code(codes.Unavailable)
	DataLoss           = Code(codes.DataLoss)
	Unauthenticated    = Code(codes.Unauthenticated)

	// Custom error codes
	Success                = Code(143)
	Failed                 = Code(144)
	WrongPassword          = Code(1005)
	ShowErrorMessage       = Code(103)
	TokenExpired           = Code(101)
	ForceLongTrip          = Code(189)
	OverMaximumNumAttempts = Code(1006)
	PinTokenExpired        = Code(1007)
)

func (c Code) String() string {
	if c >= 0 && int(c) < len(mapCodes) {
		return mapCodes[c]
	}
	if s := mapCustomCodes[c]; s != nil {
		return s.String
	}
	return "Code(" + strconv.Itoa(int(c)) + ")"
}

// ErrTODO ...
var ErrTODO = Error(Unimplemented, "TODO", nil)

// CustomCode ...
type CustomCode struct {
	StdCode        Code // Maps a custom error code to a standard grpc error code.
	String         string
	DefaultMessage string
}

var (
	mapCodes       [codes.Unauthenticated + 1]string
	mapCustomCodes map[Code]*CustomCode
)

func init() {
	mapCodes[Canceled] = "canceled"
	mapCodes[Unknown] = "unknown"
	mapCodes[InvalidArgument] = "invalid_argument"
	mapCodes[DeadlineExceeded] = "deadline_exceeded"
	mapCodes[NotFound] = "not_found"
	mapCodes[AlreadyExists] = "already_exists"
	mapCodes[PermissionDenied] = "permission_denied"
	mapCodes[Unauthenticated] = "unauthenticated"
	mapCodes[ResourceExhausted] = "resource_exhausted"
	mapCodes[FailedPrecondition] = "failed_precondition"
	mapCodes[Aborted] = "aborted"
	mapCodes[OutOfRange] = "out_of_range"
	mapCodes[Unimplemented] = "unimplemented"
	mapCodes[Internal] = "internal"
	mapCodes[Unavailable] = "unavailable"
	mapCodes[DataLoss] = "data_loss"
	mapCodes[NoError] = "ok"

	mapCustomCodes = make(map[Code]*CustomCode)
	mapCustomCodes[WrongPassword] = &CustomCode{Unauthenticated, "wrong_password", "Wrong password"}
	mapCustomCodes[TokenExpired] = &CustomCode{TokenExpired, "token_expired", "Token expired"}
	mapCustomCodes[ForceLongTrip] = &CustomCode{ForceLongTrip, "force_long_trip", "Force long trip"}
}

// APIError ...
type APIError struct {
	Code     Code // A standard grpc error code.
	XCode    Code // A custom error code, if needed.
	Err      error
	Message  string
	Original string
	Trace    bool
	Trivial  bool
	Meta     map[string]string
}

// Error ...
func (e *APIError) Error() string {
	var b strings.Builder
	_, _ = b.WriteString(e.Message)
	if e.Err != nil {
		_, _ = b.WriteString(" cause=")
		_, _ = b.WriteString(e.Err.Error())
	}
	if e.Original != "" {
		_, _ = b.WriteString(" original=")
		_, _ = b.WriteString(e.Original)
	}
	for k, v := range e.Meta {
		_ = b.WriteByte(' ')
		_, _ = b.WriteString(k)
		_ = b.WriteByte('=')
		_, _ = b.WriteString(v)
	}
	return b.String()
}

// Error ...
func Error(code Code, message string, errs ...error) *APIError {
	return newNakedError(false, code, message, errs...)
}

// Errorf ...
func Errorf(code Code, err error, message string, args ...interface{}) *APIError {
	if len(args) == 0 {
		return newError(false, code, message, err)
	}
	message = fmt.Sprintf(message, args...)
	return newError(false, code, message, err)
}

// ErrorCode ...
func ErrorCode(err error) Code {
	if err == nil {
		return NoError
	}
	if err, ok := err.(*APIError); ok {
		return err.Code
	}
	return Unknown
}

// Message ...
func Message(err error) string {
	if err == nil {
		return ""
	}
	if err, ok := err.(*APIError); ok {
		return err.Message
	}
	return err.Error()
}

// XErrorCode ...
func XErrorCode(err error) Code {
	if err == nil {
		return NoError
	}
	if err, ok := err.(*APIError); ok {
		if err.XCode != 0 {
			return err.XCode
		}
		return err.Code
	}
	return Unknown
}

// ErrorTrace ...
func ErrorTrace(code Code, message string, errs ...error) *APIError {
	return newError(true, code, message, errs...)
}

// ErrorTracef ...
func ErrorTracef(code Code, err error, message string, args ...interface{}) *APIError {
	if len(args) == 0 {
		return newError(false, code, message, err)
	}
	message = fmt.Sprintf(message, args...)
	return newError(true, code, message, err)
}

// Trace ...
func Trace(err error) *APIError {
	if xerr, ok := err.(*APIError); ok {
		xerr.Trace = true
		return xerr
	}
	if err != nil {
		return newError(true, Internal, err.Error(), err)
	}
	return newError(true, Internal, "Expected error!", nil)
}

func newNakedError(trace bool, code Code, message string, errs ...error) *APIError {
	if message == "" {
		message = code.String()
	}

	var err error
	if len(errs) > 0 {
		err = errs[0]
	}

	return &APIError{
		Err:      err,
		Code:     code,
		XCode:    code,
		Message:  message,
		Original: "",
		Trace:    trace,
	}
}

func newError(trace bool, code Code, message string, errs ...error) *APIError {
	if message == "" {
		message = code.String()
	}

	var err error
	if len(errs) > 0 {
		err = errs[0]
	}

	// Overwrite *Error
	if xerr, ok := err.(*APIError); ok {
		// Keep original message
		if xerr.Original == "" {
			xerr.Original = xerr.Message
		}
		xerr.Code = code
		xerr.XCode = code
		xerr.Message = message
		xerr.Trace = xerr.Trace || trace
		return xerr
	}

	return &APIError{
		Err:      err,
		Code:     code,
		XCode:    code,
		Message:  message,
		Original: "",
		Trace:    trace,
	}
}

// IsValidStandardErrorCode ...
func IsValidStandardErrorCode(c Code) bool {
	return c >= 0 && int(c) < len(mapCodes)
}

func ErrorToCode(err error) int {
	cmErr, ok := err.(*APIError)
	if !ok {
		return http.StatusInternalServerError
	}

	switch cmErr.Code {
	case NotFound, FailedPrecondition:
		return http.StatusBadRequest
	default:
	}

	return http.StatusInternalServerError
}

// ToGRPCError ...
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if xerr, ok := err.(*APIError); ok {
		return ErrorGRPC(codes.Code(xerr.Code), xerr.Message)
	}
	code := grpc.Code(err)
	if code == codes.Unknown {
		return ErrorGRPC(codes.Internal, "Internal Server Error")
	}
	return err
}

func ErrorGRPC(code codes.Code, msg string, details ...proto.Message) error {
	if msg == "" {
		switch code {
		case codes.NotFound:
			msg = "Not Found"
		case codes.InvalidArgument:
			msg = "Invalid Argument"
		case codes.Internal:
			msg = "Internal Server Error"
		case codes.Unauthenticated:
			msg = "Unauthenticated"
		case codes.PermissionDenied:
			msg = "Permission Denied"
		default:
			msg = "Unknown"
		}
	}
	s := &spb.Status{
		Code:    int32(code),
		Message: msg,
	}
	if len(details) > 0 {
		ds := make([]*any.Any, len(details))
		for i, d := range details {
			any, err := anypb.New(d)
			if err != nil {
				debug.PrintStack()
				log.Println("Unable to marshal any")
				ds[i], _ = anypb.New(status.New(codes.Internal, "Unable to marshal to grpc.Any").Proto())
			} else {
				ds[i] = any
			}
		}
		s.Details = ds
	}
	return status.ErrorProto(s)
}
