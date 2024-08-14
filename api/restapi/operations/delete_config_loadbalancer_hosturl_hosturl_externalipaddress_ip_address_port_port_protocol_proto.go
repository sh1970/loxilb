// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoHandlerFunc turns a function with the right signature into a delete config loadbalancer hosturl hosturl externalipaddress IP address port port protocol proto handler
type DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoHandlerFunc func(DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoParams) middleware.Responder

// Handle executing the request and returning a response
func (fn DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoHandlerFunc) Handle(params DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoParams) middleware.Responder {
	return fn(params)
}

// DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoHandler interface for that can handle valid delete config loadbalancer hosturl hosturl externalipaddress IP address port port protocol proto params
type DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoHandler interface {
	Handle(DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoParams) middleware.Responder
}

// NewDeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProto creates a new http.Handler for the delete config loadbalancer hosturl hosturl externalipaddress IP address port port protocol proto operation
func NewDeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProto(ctx *middleware.Context, handler DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoHandler) *DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProto {
	return &DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProto{Context: ctx, Handler: handler}
}

/*
	DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProto swagger:route DELETE /config/loadbalancer/hosturl/{hosturl}/externalipaddress/{ip_address}/port/{port}/protocol/{proto} deleteConfigLoadbalancerHosturlHosturlExternalipaddressIpAddressPortPortProtocolProto

# Delete an existing Load balancer service

Delete an existing load balancer service with .
*/
type DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProto struct {
	Context *middleware.Context
	Handler DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoHandler
}

func (o *DeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProto) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewDeleteConfigLoadbalancerHosturlHosturlExternalipaddressIPAddressPortPortProtocolProtoParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
