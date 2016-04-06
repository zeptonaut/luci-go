// Code generated by svcdec; DO NOT EDIT

package tokenserver

import (
	proto "github.com/golang/protobuf/proto"
	context "golang.org/x/net/context"
)

type DecoratedServiceAccounts struct {
	// Service is the service to decorate.
	Service ServiceAccountsServer
	// Prelude is called in each method before forwarding the call to Service.
	// If Prelude returns an error, it is returned without forwarding the call.
	Prelude func(c context.Context, methodName string, req proto.Message) (context.Context, error)
}

func (s *DecoratedServiceAccounts) CreateServiceAccount(c context.Context, req *CreateServiceAccountRequest) (*CreateServiceAccountResponse, error) {
	c, err := s.Prelude(c, "CreateServiceAccount", req)
	if err != nil {
		return nil, err
	}
	return s.Service.CreateServiceAccount(c, req)
}