// Code generated by go-swagger; DO NOT EDIT.

package health

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// GetLivenessHandlerFunc turns a function with the right signature into a get liveness handler
type GetLivenessHandlerFunc func(GetLivenessParams) middleware.Responder

// Handle executing the request and returning a response
func (fn GetLivenessHandlerFunc) Handle(params GetLivenessParams) middleware.Responder {
	return fn(params)
}

// GetLivenessHandler interface for that can handle valid get liveness params
type GetLivenessHandler interface {
	Handle(GetLivenessParams) middleware.Responder
}

// NewGetLiveness creates a new http.Handler for the get liveness operation
func NewGetLiveness(ctx *middleware.Context, handler GetLivenessHandler) *GetLiveness {
	return &GetLiveness{Context: ctx, Handler: handler}
}

/*
	GetLiveness swagger:route GET /liveness health getLiveness

Check if Goliac is healthy
*/
type GetLiveness struct {
	Context *middleware.Context
	Handler GetLivenessHandler
}

func (o *GetLiveness) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewGetLivenessParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
