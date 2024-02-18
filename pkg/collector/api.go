package collector

import (
	"fmt"

	"github.com/samber/lo"

	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
)

type ApiCollectorServer struct {
	requirements *ServiceRequirements
	apispb.UnimplementedApiServer
}

func (s *ApiCollectorServer) Serve(stream apispb.Api_ServeServer) error {
	s.requirements.resourceLock.Lock()
	defer s.requirements.resourceLock.Unlock()

	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	registrationRequest := msg.GetRegistrationRequest()

	if registrationRequest == nil {
		return fmt.Errorf("first message must be a registration request")
	}

	existingRoute, found := lo.Find(s.requirements.routes[registrationRequest.Api], func(item *apispb.RegistrationRequest) bool {
		return len(lo.Intersect(item.Methods, registrationRequest.Methods)) > 0 && item.Path == registrationRequest.Path
	})

	if found {
		conflictingMethods := lo.Intersect(existingRoute.Methods, registrationRequest.Methods)
		for _, conflictingMethod := range conflictingMethods {
			s.requirements.errors = append(s.requirements.errors, fmt.Errorf("%s: %s already registered for API '%s'", conflictingMethod, existingRoute.Path, existingRoute.Api))
		}
	} else {
		s.requirements.routes[registrationRequest.Api] = append(s.requirements.routes[registrationRequest.Api], registrationRequest)
	}

	return stream.Send(&apispb.ServerMessage{
		Content: &apispb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &apispb.RegistrationResponse{},
		},
	})
}
