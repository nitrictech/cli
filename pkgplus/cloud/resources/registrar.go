package resources

import (
	"fmt"
	"slices"
	"sync"
)

type ResourceRegistrar[R any] struct {
	lock      sync.RWMutex
	resources map[ResourceName]*struct {
		requestingServices []string
		resource           *R
	}
}

func (r *ResourceRegistrar[R]) isAlreadyRegistered(name string, requestingService string) bool {
	_, exists := r.resources[name]
	if exists {
		duplicate := slices.Contains(r.resources[name].requestingServices, requestingService)
		if !duplicate {
			return true
		}
	}

	return false
}

func (r *ResourceRegistrar[R]) Register(name string, requestingService string, resource *R) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, exists := r.resources[name]
	if exists {
		if r.isAlreadyRegistered(name, requestingService) {
			return fmt.Errorf("resource %s registered multiple times for service %s", name, requestingService)
		}

		// already registered, by another service, add this service to the list
		r.resources[name].requestingServices = append(r.resources[name].requestingServices, requestingService)
		return nil
	}

	// new resource, register it
	r.resources[name] = &struct {
		requestingServices []string
		resource           *R
	}{
		requestingServices: []string{requestingService},
		resource:           resource,
	}
	return nil
}

func (r *ResourceRegistrar[R]) Get(name string) *R {
	r.lock.RLock()
	defer r.lock.RUnlock()

	registration, ok := r.resources[name]
	if !ok {
		return nil
	}
	return registration.resource
}

func (r *ResourceRegistrar[R]) GetRequestingServices(name string) []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	registration, ok := r.resources[name]
	if !ok {
		return []string{}
	}
	return registration.requestingServices
}

// ClearRequestingService - Remove a requesting service from all resources, if it was the only requestor for a resource, remove the resource
func (r *ResourceRegistrar[R]) ClearRequestingService(requestingService string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for name, registration := range r.resources {
		for i, service := range registration.requestingServices {
			if service == requestingService {
				// TODO: are the indexes here correct or should it be i, i+1?
				registration.requestingServices = slices.Delete(registration.requestingServices, i, i)
			}
		}
		if len(registration.requestingServices) == 0 {
			delete(r.resources, name)
		}
	}
}
