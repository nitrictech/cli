package resources

import (
	"fmt"
	"slices"
	"sync"
)

type ResourceRegister[R any] struct {
	RequestingServices []string
	Resource           *R
}

type ResourceRegistrar[R any] struct {
	lock      sync.RWMutex
	resources map[ResourceName]*ResourceRegister[R]
}

func (r *ResourceRegistrar[R]) isAlreadyRegistered(name string, requestingService string) bool {
	_, exists := r.resources[name]
	if exists {
		duplicate := slices.Contains(r.resources[name].RequestingServices, requestingService)
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
		r.resources[name].RequestingServices = append(r.resources[name].RequestingServices, requestingService)
		return nil
	}

	// new resource, register it
	r.resources[name] = &ResourceRegister[R]{
		RequestingServices: []string{requestingService},
		Resource:           resource,
	}
	return nil
}

func (r *ResourceRegistrar[R]) Get(resourceName string) *R {
	r.lock.RLock()
	defer r.lock.RUnlock()

	registration, ok := r.resources[resourceName]
	if !ok {
		return nil
	}
	return registration.Resource
}

func (r *ResourceRegistrar[R]) GetAll() map[ResourceName]*ResourceRegister[R] {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.resources
}

func (r *ResourceRegistrar[R]) GetRequestingServices(name string) []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	registration, ok := r.resources[name]
	if !ok {
		return []string{}
	}
	return registration.RequestingServices
}

// ClearRequestingService - Remove a requesting service from all resources, if it was the only requestor for a resource, the resource is also removed
func (r *ResourceRegistrar[R]) ClearRequestingService(requestingService string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for name, registration := range r.resources {
		for i, service := range registration.RequestingServices {
			if service == requestingService {
				// TODO investigate slice bounds error when refreshing code, could just append to a new slice
				registration.RequestingServices = slices.Delete(registration.RequestingServices, i, i+1)
			}
		}
		if len(registration.RequestingServices) == 0 {
			delete(r.resources, name)
		}
	}
}

func NewResourceRegistrar[R any]() *ResourceRegistrar[R] {
	return &ResourceRegistrar[R]{
		resources: make(map[string]*ResourceRegister[R], 0),
	}
}
