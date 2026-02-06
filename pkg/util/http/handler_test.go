package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeHTTPHandlerFunc_NilResponse(t *testing.T) {
	require := require.New(t)

	handlerFn := MakeHTTPHandlerFunc(func(ctx *Context) (any, error) {
		return nil, nil
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/proxies?status=offline", nil)
	handlerFn.ServeHTTP(recorder, req)

	require.Equal(http.StatusNoContent, recorder.Code,
		"nil response should produce 204 No Content so that clients do not attempt to parse an empty body as JSON")
	require.Empty(recorder.Body.String(), "body should be empty for 204 responses")
}

func TestMakeHTTPHandlerFunc_StructResponse(t *testing.T) {
	require := require.New(t)

	type payload struct {
		Count int `json:"count"`
	}

	handlerFn := MakeHTTPHandlerFunc(func(ctx *Context) (any, error) {
		return payload{Count: 42}, nil
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	handlerFn.ServeHTTP(recorder, req)

	require.Equal(http.StatusOK, recorder.Code)
	require.Equal("application/json", recorder.Header().Get("Content-Type"))

	var result payload
	err := json.NewDecoder(recorder.Body).Decode(&result)
	require.NoError(err)
	require.Equal(42, result.Count)
}

func TestMakeHTTPHandlerFunc_ErrorResponse(t *testing.T) {
	require := require.New(t)

	handlerFn := MakeHTTPHandlerFunc(func(ctx *Context) (any, error) {
		return nil, NewError(http.StatusBadRequest, "invalid parameter")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	handlerFn.ServeHTTP(recorder, req)

	require.Equal(http.StatusBadRequest, recorder.Code)

	var resp GeneralResponse
	err := json.NewDecoder(recorder.Body).Decode(&resp)
	require.NoError(err)
	require.Equal(http.StatusBadRequest, resp.Code)
	require.Contains(resp.Msg, "invalid parameter")
}
