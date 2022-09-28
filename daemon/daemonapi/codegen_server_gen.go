// Package daemonapi provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.11.0 DO NOT EDIT.
package daemonapi

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /object/config)
	GetObjectConfig(w http.ResponseWriter, r *http.Request, params GetObjectConfigParams)

	// (GET /object/file)
	GetObjectFile(w http.ResponseWriter, r *http.Request, params GetObjectFileParams)

	// (POST /object/monitor)
	PostObjectMonitor(w http.ResponseWriter, r *http.Request)

	// (GET /object/selector)
	GetObjectSelector(w http.ResponseWriter, r *http.Request, params GetObjectSelectorParams)

	// (POST /object/status)
	PostObjectStatus(w http.ResponseWriter, r *http.Request)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.HandlerFunc) http.HandlerFunc

// GetObjectConfig operation middleware
func (siw *ServerInterfaceWrapper) GetObjectConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetObjectConfigParams

	// ------------- Required query parameter "path" -------------
	if paramValue := r.URL.Query().Get("path"); paramValue != "" {

	} else {
		siw.ErrorHandlerFunc(w, r, &RequiredParamError{ParamName: "path"})
		return
	}

	err = runtime.BindQueryParameter("form", true, true, "path", r.URL.Query(), &params.Path)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "path", Err: err})
		return
	}

	// ------------- Optional query parameter "evaluate" -------------
	if paramValue := r.URL.Query().Get("evaluate"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "evaluate", r.URL.Query(), &params.Evaluate)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "evaluate", Err: err})
		return
	}

	// ------------- Optional query parameter "impersonate" -------------
	if paramValue := r.URL.Query().Get("impersonate"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "impersonate", r.URL.Query(), &params.Impersonate)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "impersonate", Err: err})
		return
	}

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetObjectConfig(w, r, params)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetObjectFile operation middleware
func (siw *ServerInterfaceWrapper) GetObjectFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetObjectFileParams

	// ------------- Required query parameter "path" -------------
	if paramValue := r.URL.Query().Get("path"); paramValue != "" {

	} else {
		siw.ErrorHandlerFunc(w, r, &RequiredParamError{ParamName: "path"})
		return
	}

	err = runtime.BindQueryParameter("form", true, true, "path", r.URL.Query(), &params.Path)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "path", Err: err})
		return
	}

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetObjectFile(w, r, params)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// PostObjectMonitor operation middleware
func (siw *ServerInterfaceWrapper) PostObjectMonitor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PostObjectMonitor(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetObjectSelector operation middleware
func (siw *ServerInterfaceWrapper) GetObjectSelector(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetObjectSelectorParams

	// ------------- Required query parameter "selector" -------------
	if paramValue := r.URL.Query().Get("selector"); paramValue != "" {

	} else {
		siw.ErrorHandlerFunc(w, r, &RequiredParamError{ParamName: "selector"})
		return
	}

	err = runtime.BindQueryParameter("form", true, true, "selector", r.URL.Query(), &params.Selector)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "selector", Err: err})
		return
	}

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetObjectSelector(w, r, params)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// PostObjectStatus operation middleware
func (siw *ServerInterfaceWrapper) PostObjectStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PostObjectStatus(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

type UnescapedCookieParamError struct {
	ParamName string
	Err       error
}

func (e *UnescapedCookieParamError) Error() string {
	return fmt.Sprintf("error unescaping cookie parameter '%s'", e.ParamName)
}

func (e *UnescapedCookieParamError) Unwrap() error {
	return e.Err
}

type UnmarshalingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshalingParamError) Error() string {
	return fmt.Sprintf("Error unmarshaling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshalingParamError) Unwrap() error {
	return e.Err
}

type RequiredParamError struct {
	ParamName string
}

func (e *RequiredParamError) Error() string {
	return fmt.Sprintf("Query argument %s is required, but not found", e.ParamName)
}

type RequiredHeaderError struct {
	ParamName string
	Err       error
}

func (e *RequiredHeaderError) Error() string {
	return fmt.Sprintf("Header parameter %s is required, but not found", e.ParamName)
}

func (e *RequiredHeaderError) Unwrap() error {
	return e.Err
}

type InvalidParamFormatError struct {
	ParamName string
	Err       error
}

func (e *InvalidParamFormatError) Error() string {
	return fmt.Sprintf("Invalid format for parameter %s: %s", e.ParamName, e.Err.Error())
}

func (e *InvalidParamFormatError) Unwrap() error {
	return e.Err
}

type TooManyValuesForParamError struct {
	ParamName string
	Count     int
}

func (e *TooManyValuesForParamError) Error() string {
	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{})
}

type ChiServerOptions struct {
	BaseURL          string
	BaseRouter       chi.Router
	Middlewares      []MiddlewareFunc
	ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, r chi.Router) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseRouter: r,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, r chi.Router, baseURL string) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseURL:    baseURL,
		BaseRouter: r,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options ChiServerOptions) http.Handler {
	r := options.BaseRouter

	if r == nil {
		r = chi.NewRouter()
	}
	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandlerFunc:   options.ErrorHandlerFunc,
	}

	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/object/config", wrapper.GetObjectConfig)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/object/file", wrapper.GetObjectFile)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/object/monitor", wrapper.PostObjectMonitor)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/object/selector", wrapper.GetObjectSelector)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/object/status", wrapper.PostObjectStatus)
	})

	return r
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/9RWT2/bPgz9KgZ/v6NWpy128XHDNuwwrECPRTAoNpOosyVVooNlhb/7IMn/EstNurUD",
	"dktMiu+RjyL1CLmqtJIoyUL2CJobXiGh8f8eajT7r6t7zOmG09Z9KtDmRmgSSkIGytsS7YwMhPvkzwAD",
	"ySuEDFqTwYdaGCwgI1MjA5tvseIuIO2187NkhNxA07Ax6i2WmJMys8i2c4ijj8znM2g6o6/BB2MCvjZK",
	"oyGB/nOuCpyy8s6JtzFYK1NxggyEpOsrYB2QkIQbNNAwqNBavpkN1JlZpEpDPnfQAnbuy4bBjbIUavhF",
	"SUGxFDalWvHyG/7QmFOkECyIF9VojO69lj3HoAwccLglTrWdUpgBYGD7A/8bXEMG/6VDo6atPKmQlrjM",
	"sQ0fJdbHijE8CjDhx3dclKdYtPEbBmujfqJ0/r32BSd8Q6KKiMjgu5BFNH3l+4A/A1rt0PDyGQe0UTth",
	"hZI4prBSqkQunUOtHffi3GyOah8qN/Aa4vVlOuQQkyf8eq/kWmym4hScxvd3OFZ5hufznsH9KEqcR+1j",
	"r/YUFfe5LMbVq1ovDzZfmZu566Mm01MQVifv0yjqUBZuDN8f3snIxBRyrbxJUOmpapR2lydcC2CwQ2PD",
	"YLu+WFwsQoujdMbuU5g2PnzLI8174TfoB5TTgbub8bmADD5hO13aBmEH2+sunurgkh5vt4YdT2Hc8bLm",
	"Xt7YdhmZJ9ukv0jTqKLSaKySnDChLSZtGKFkwm0iwyyP4Y0OwlMLbOmayWolbWjZq8UirCxJKH0hudal",
	"yD1mem+VH1lDvNMt0lbcK3+YnK3zHK2bxc7WKblur9LTOvoL98cqvn72nufZuVej/atsJP/pqg7DAC29",
	"U8X+xchPcSI5VAcUhgdTE6/qTAFc1DWvS3ox8uEZFiFcy/CAwSLBzmdUfTuagU933+3wVvztDuxj/IUu",
	"HLDO7cRhgJ9qxPY99Np92L/bJgk4gkn3xh+T+Vc70u1PNLuupWpTQgZbIp2laalyXm6Vpezy6vJt6rZi",
	"s2x+BQAA//+AzkL1mw0AAA==",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	var res = make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	var resolvePath = PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		var pathToFile = url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
