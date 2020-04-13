// Package server is a layer responsible for request/response on HTTP level, it delegates to backend for everything else.
package server

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/plumber-cd/terraform-backend-git/backend"
	"github.com/plumber-cd/terraform-backend-git/types"
)

// HandleFunc main function responsible for routing
func HandleFunc(response http.ResponseWriter, request *http.Request) {
	handler := handler{
		Request:  request,
		Response: response,
	}

	metadata, err := backend.ParseMetadata(request)
	if err != nil {
		handler.clientError(err)
		return
	}

	storageClient, err := backend.GetStorageClient(metadata)
	if err != nil {
		handler.clientError(err)
		return
	}

	if err := storageClient.ParseMetadataParams(request, metadata); err != nil {
		handler.clientError(err)
		return
	}

	defer storageClient.Disconnect(metadata.Params)
	if err := storageClient.Connect(metadata.Params); err != nil {
		handler.serverError(err)
		return
	}

	switch request.Method {
	case "LOCK":
		log.Printf("Locking state in %s", metadata.Params.String())

		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			handler.clientError(err)
			return
		}

		if err := backend.LockState(metadata, storageClient, body); err != nil {
			handler.serverError(err)
			return
		}

		response.WriteHeader(http.StatusOK)
	case "UNLOCK":
		log.Printf("Unlocking state in %s", metadata.Params.String())

		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			handler.clientError(err)
			return
		}

		if err := backend.UnLockState(metadata, storageClient, body); err != nil {
			handler.serverError(err)
			return
		}

		response.WriteHeader(http.StatusOK)
	case http.MethodGet:
		log.Printf("Getting state from %s", metadata.Params.String())

		state, err := backend.GetState(metadata, storageClient)
		if err != nil {
			handler.serverError(err)
			return
		}

		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		response.Write(state)
	case http.MethodPost:
		log.Printf("Saving state to %s", metadata.Params.String())

		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			handler.clientError(err)
			return
		}

		if err := backend.UpdateState(metadata, storageClient, body); err != nil {
			handler.serverError(err)
			return
		}

		response.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		// TODO: It doesn't looks like TF ever uses this method actually... couldn't find testing scenario that works.
		// "terraform destroy" just writes a new state with empty list of resources.
		// "terraform state pull|push" never deletes the old state locations either.
		// But, based on the code in https://github.com/hashicorp/terraform/blob/master/backend/remote-state/http/client.go,
		// the code is broken anyway - it doesn't send the Lock ID for this call in HTTP request params.
		// Even if TF will ever make this call - it'll fail since request will never have a Lock ID in it,
		// and this is a write type operation.

		log.Printf("Deleting state from %s", metadata.Params.String())

		if err := backend.DeleteState(metadata, storageClient); err != nil {
			handler.serverError(err)
			return
		}

		response.WriteHeader(http.StatusOK)
	default:
		handler.clientError(errors.New("Unknown method: " + request.Method))
	}
}

// handler just a handy struct to store request and response
type handler struct {
	Request  *http.Request
	Response http.ResponseWriter
}

// serverError handle the error by default assuming it was a server side error
func (handler *handler) serverError(err error) {
	handler.responseError(http.StatusInternalServerError, "500 - Internal Server Error", err)
}

// clientError handle the error by default assuming it was a client side error
func (handler *handler) clientError(err error) {
	handler.responseError(http.StatusBadRequest, "400 - Bad Request", err)
}

// responseError is a handler that will try to read known errors and formulate approapriate responses to them
// If error was unknown, just use defaultCode and defaultResponse error message.
func (handler *handler) responseError(defaultCode int, defaultResponse string, actualErr error) {
	log.Printf("%s", actualErr)
	switch actualErr.(type) {
	case *types.ErrLocked:
		handler.Response.WriteHeader(http.StatusConflict)
		handler.Response.Write(actualErr.(*types.ErrLocked).Lock)
	default:
		switch actualErr {
		case types.ErrLockMissing:
			handler.Response.WriteHeader(http.StatusPreconditionRequired)
			handler.Response.Write([]byte("428 - Locking Required"))
		case types.ErrStateDidNotExisted:
			handler.Response.WriteHeader(http.StatusNoContent)
		default:
			handler.Response.WriteHeader(defaultCode)
			handler.Response.Write([]byte(defaultResponse))
		}
	}
}
