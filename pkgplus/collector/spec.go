package collector

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/samber/lo"

	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
	"github.com/nitrictech/pearls/pkg/tui/view"
	"github.com/samber/lo"
)

type ProjectErrors struct {
	errors []error
}

func (pe *ProjectErrors) Add(err error) {
	pe.errors = append(pe.errors, err)
}

func (pe ProjectErrors) Error() error {
	if len(pe.errors) > 0 {
		errorView := view.New()

		errorView.AddRow(view.NewFragment("Errors found in project:"))

		for _, err := range pe.errors {
			errorView.AddRow(view.NewFragment(fmt.Sprintf("- %s", err.Error())).WithStyle(lipgloss.NewStyle().MarginLeft(2)))
		}

		return fmt.Errorf(errorView.Render())
	}

	return nil
}

// buildBucketRequirements gathers and deduplicates all bucket requirements
func buildBucketRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	for _, serviceRequirements := range allServiceRequirements {
		for bucketName := range serviceRequirements.buckets {
			notifications := []*deploymentspb.BucketNotificationTarget{}

			for _, v := range serviceRequirements.listeners {
				notifications = append(notifications, &deploymentspb.BucketNotificationTarget{
					Config: v,
					Target: &deploymentspb.BucketNotificationTarget_ExecutionUnit{
						ExecutionUnit: serviceRequirements.serviceName,
					},
				})
			}

			res, exists := lo.Find(resources, func(item *deploymentspb.Resource) bool {
				return item.Name == bucketName
			})

			if exists {
				// add the listeners to the bucket configuration
				res.GetBucket().Notifications = append(res.GetBucket().Notifications, notifications...)
			} else {
				res := &deploymentspb.Resource{
					Name: bucketName,
					Type: resourcespb.ResourceType_Bucket,
					Config: &deploymentspb.Resource_Bucket{
						Bucket: &deploymentspb.Bucket{
							Notifications: notifications,
						},
					},
				}
				resources = append(resources, res)
			}
		}
	}

	return resources, nil
}

// buildHttpRequirements gathers and deduplicates all http requirements
func buildHttpRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	for _, serviceRequirements := range allServiceRequirements {
		if serviceRequirements.proxy != nil {
			resources = append(resources, &deploymentspb.Resource{
				Name: serviceRequirements.serviceName,
				Type: resourcespb.ResourceType_Http,
				Config: &deploymentspb.Resource_Http{
					Http: &deploymentspb.Http{
						Target: &deploymentspb.HttpTarget{
							Target: &deploymentspb.HttpTarget_ExecutionUnit{
								ExecutionUnit: serviceRequirements.serviceName,
							},
						},
					},
				},
			})
		}
	}

	return resources, nil
}

// buildTopicRequirements gathers and deduplicates all topic requirements
func buildTopicRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	for _, serviceRequirements := range allServiceRequirements {
		for topicName := range serviceRequirements.topics {
			res, exists := lo.Find(resources, func(item *deploymentspb.Resource) bool {
				return item.Name == topicName
			})

			if !exists {
				res = &deploymentspb.Resource{
					Name: topicName,
					Type: resourcespb.ResourceType_Topic,
					Config: &deploymentspb.Resource_Topic{
						Topic: &deploymentspb.Topic{
							Subscriptions: []*deploymentspb.SubscriptionTarget{},
						},
					},
				}
				resources = append(resources, res)
			}

			if len(serviceRequirements.subscriptions[topicName]) > 0 {
				res.GetTopic().Subscriptions = append(res.GetTopic().Subscriptions, &deploymentspb.SubscriptionTarget{
					Target: &deploymentspb.SubscriptionTarget_ExecutionUnit{
						ExecutionUnit: serviceRequirements.serviceName,
					},
				})
			}
		}
	}

	return resources, nil
}

// buildSecretRequirements gathers and deduplicates all secret requirements
func buildSecretRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	for _, serviceRequirements := range allServiceRequirements {
		for secretName := range serviceRequirements.secrets {
			_, exists := lo.Find(resources, func(item *deploymentspb.Resource) bool {
				return item.Name == secretName
			})

			if !exists {
				res := &deploymentspb.Resource{
					Name: secretName,
					Type: resourcespb.ResourceType_Secret,
					Config: &deploymentspb.Resource_Secret{
						Secret: &deploymentspb.Secret{},
					},
				}
				// add the listeners to the bucket configuration
				resources = append(resources, res)
			}
		}
	}

	return resources, nil
}

func ensureOneTrailingSlash(p string) string {
	if len(p) > 0 && string(p[len(p)-1]) == "/" {
		return p
	}

	return p + "/"
}

// openAPIPathAndParams splits a path into an OpenAPI 3 path and a list of OpenAPI 3 parameters
// this is done by splitting the path on '/' and looking for path parameters (e.g. /foo/:bar)
// and replacing them with OpenAPI 3 parameters (e.g. /foo/{bar})
func openAPIPathAndParams(workerPath string) (string, openapi3.Parameters) {
	normalizedPath := ""
	params := make(openapi3.Parameters, 0)

	for _, p := range strings.Split(workerPath, "/") {
		if strings.HasPrefix(p, ":") {
			paramName := strings.Replace(p, ":", "", -1)

			params = append(params, &openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					In:       "path",
					Name:     paramName,
					Required: true,
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "string",
						},
					},
				},
			})
			normalizedPath = ensureOneTrailingSlash(normalizedPath + "{" + paramName + "}")
		} else {
			normalizedPath = ensureOneTrailingSlash(normalizedPath + p)
		}
	}
	// trim off trailing slash
	if normalizedPath != "/" {
		normalizedPath = strings.TrimSuffix(normalizedPath, "/")
	}

	return normalizedPath, params
}

var notAlphaNumeric, _ = regexp.Compile("[^a-zA-Z0-9]+")

// buildApiRequirements gathers and deduplicates all api requirements
func buildApiRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	apis := map[string]*openapi3.T{}

	for _, serviceRequirements := range allServiceRequirements {
		for apiName, apiResource := range serviceRequirements.apis {
			api, exists := apis[apiName]
			if !exists {
				api = &openapi3.T{
					Paths: make(openapi3.Paths),
					Info: &openapi3.Info{
						Title:   apiName,
						Version: "v1",
					},
					OpenAPI: "3.0.1",
					Components: &openapi3.Components{
						SecuritySchemes: make(openapi3.SecuritySchemes),
					},
				}

				apis[apiName] = api
			}

			// Add security schemes to the documement
			for schemeName, securityScheme := range serviceRequirements.apiSecurityDefinition[apiName] {
				switch securityScheme.GetDefinition().(type) {
				case *resourcespb.ApiSecurityDefinitionResource_Oidc:
					issuerUrl := securityScheme.GetOidc().GetIssuer()

					oidSec := openapi3.NewOIDCSecurityScheme(issuerUrl)
					oidSec.Extensions = map[string]interface{}{
						"x-nitric-audiences": securityScheme.GetOidc().GetAudiences(),
					}
					oidSec.Name = schemeName

					api.Components.SecuritySchemes[securityScheme.GetApiName()] = &openapi3.SecuritySchemeRef{
						Value: oidSec,
					}
				default:
					return nil, fmt.Errorf("unknown security definition type: %T", securityScheme.GetDefinition())
				}
			}

			// apply top level security rules
			for schemeName, scopes := range apiResource.Security {
				api.Security.With(openapi3.SecurityRequirement{
					schemeName: scopes.Scopes,
				})
			}

			// apply route level security rules
			for _, route := range serviceRequirements.routes[apiName] {
				normalizedPath, params := openAPIPathAndParams(route.Path)
				pathItem := api.Paths.Find(normalizedPath)

				if pathItem == nil {
					// Add the parameters at the path level
					pathItem = &openapi3.PathItem{
						Parameters: params,
					}
					// Add the path item to the document
					api.Paths[normalizedPath] = pathItem
				}

				for _, method := range route.Methods {
					if pathItem.Operations() != nil && pathItem.Operations()[method] != nil {
						operation := pathItem.Operations()[method]

						existingServiceName := operation.Extensions["x-nitric-target"].(map[string]string)["name"]

						existingService, _ := lo.Find(allServiceRequirements, func(item *ServiceRequirements) bool {
							return existingServiceName == item.serviceName
						})

						projectErrors.Add(fmt.Errorf("service %s attempted to register duplicate route %s: %s for API '%s' which is already defined in service %s", serviceRequirements.serviceFile, method, route.Path, apiName, existingService.serviceFile))
						continue
					}

					exts := map[string]interface{}{
						"x-nitric-target": map[string]string{
							"type": "function",
							"name": serviceRequirements.serviceName,
						},
					}

					var sr *openapi3.SecurityRequirements = nil

					if route.GetOptions() != nil {
						if route.GetOptions().SecurityDisabled {
							sr = &openapi3.SecurityRequirements{}
						} else if len(route.GetOptions().Security) > 0 {
							sr = &openapi3.SecurityRequirements{}
							if !route.GetOptions().SecurityDisabled {
								for key, scopes := range route.GetOptions().GetSecurity() {
									sr.With(openapi3.SecurityRequirement{
										key: scopes.Scopes,
									})
								}
							}
						}
					}

					pathItem.SetOperation(method, &openapi3.Operation{
						OperationID: strings.ToLower(notAlphaNumeric.ReplaceAllString(normalizedPath+method, "")),
						Responses:   openapi3.NewResponses(),
						Extensions:  exts,
						Security:    sr,
					})
				}
			}
		}
	}

	for apiName, api := range apis {
		openApiJsonDoc, err := api.MarshalJSON()
		if err != nil {
			return nil, err
		}

		resources = append(resources, &deploymentspb.Resource{
			Name: apiName,
			Type: resourcespb.ResourceType_Api,
			Config: &deploymentspb.Resource_Api{
				Api: &deploymentspb.Api{
					Document: &deploymentspb.Api_Openapi{
						Openapi: string(openApiJsonDoc),
					},
				},
			},
		})
	}

	return resources, nil
}

// buildWebsocketRequirements gathers and deduplicates all websocket requirements
func buildWebsocketRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	for _, serviceRequirements := range allServiceRequirements {
		for socketName, registrations := range serviceRequirements.websockets {
			res, exists := lo.Find(resources, func(item *deploymentspb.Resource) bool {
				return item.Name == socketName
			})

			if !exists {
				res = &deploymentspb.Resource{
					Name: socketName,
					Type: resourcespb.ResourceType_Websocket,
					Config: &deploymentspb.Resource_Websocket{
						Websocket: &deploymentspb.Websocket{},
					},
				}
				resources = append(resources, res)
			}

			for _, registration := range registrations {
				switch registration.EventType {
				case *websocketspb.WebsocketEventType_Connect.Enum():
					res.GetWebsocket().ConnectTarget = &deploymentspb.WebsocketTarget{
						Target: &deploymentspb.WebsocketTarget_ExecutionUnit{
							ExecutionUnit: serviceRequirements.serviceName,
						},
					}
				case *websocketspb.WebsocketEventType_Disconnect.Enum():
					res.GetWebsocket().DisconnectTarget = &deploymentspb.WebsocketTarget{
						Target: &deploymentspb.WebsocketTarget_ExecutionUnit{
							ExecutionUnit: serviceRequirements.serviceName,
						},
					}
				case *websocketspb.WebsocketEventType_Message.Enum():
					res.GetWebsocket().MessageTarget = &deploymentspb.WebsocketTarget{
						Target: &deploymentspb.WebsocketTarget_ExecutionUnit{
							ExecutionUnit: serviceRequirements.serviceName,
						},
					}
				}
			}
		}
	}

	// loop over all websockets and make sure all methods are handled
	for _, websocketResource := range resources {
		ws := websocketResource.GetWebsocket()

		if ws.ConnectTarget == nil {
			projectErrors.Add(fmt.Errorf("missing connect handler for websocket %s", websocketResource.Name))
		}

		if ws.DisconnectTarget == nil {
			projectErrors.Add(fmt.Errorf("missing disconnect handler for websocket %s", websocketResource.Name))
		}

		if ws.MessageTarget == nil {
			projectErrors.Add(fmt.Errorf("missing message handler for websocket %s", websocketResource.Name))
		}
	}

	return resources, nil
}

// buildScheduleRequirements gathers all schedule requirements, erroring on duplicate schedule names
func buildScheduleRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	for _, serviceRequirements := range allServiceRequirements {
		for scheduleName, scheduleConfig := range serviceRequirements.schedules {
			_, exists := lo.Find(resources, func(item *deploymentspb.Resource) bool {
				return item.Name == scheduleName
			})

			if !exists {
				schedule := &deploymentspb.Schedule{}

				switch t := scheduleConfig.Cadence.(type) {
				case *schedulespb.RegistrationRequest_Cron:
					schedule.Cadence = &deploymentspb.Schedule_Cron{
						Cron: &deploymentspb.ScheduleCron{
							Expression: t.Cron.Expression,
						},
					}
				case *schedulespb.RegistrationRequest_Every:
					schedule.Cadence = &deploymentspb.Schedule_Every{
						Every: &deploymentspb.ScheduleEvery{
							Rate: t.Every.Rate,
						},
					}
				}

				schedule.Target = &deploymentspb.ScheduleTarget{
					Target: &deploymentspb.ScheduleTarget_ExecutionUnit{
						ExecutionUnit: serviceRequirements.serviceName,
					},
				}

				res := &deploymentspb.Resource{
					Name: scheduleName,
					Type: resourcespb.ResourceType_Schedule,
					Config: &deploymentspb.Resource_Schedule{
						Schedule: schedule,
					},
				}
				resources = append(resources, res)
			} else {
				projectErrors.Add(fmt.Errorf("multiple schedules registered with name '%s', schedule names must be unique", scheduleName))
			}
		}
	}

	return resources, nil
}

// policyResourceName generates a unique name for a policy resource by hashing the policy document
func policyResourceName(policy *resourcespb.PolicyResource) (string, error) {
	policyDoc, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	hasher := md5.New()
	hasher.Write(policyDoc)

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// buildCollectionsRequirements gathers and deduplicates all collection requirements
func buildCollectionsRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	for _, serviceRequirements := range allServiceRequirements {
		for collectionName := range serviceRequirements.collections {
			_, exists := lo.Find(resources, func(item *deploymentspb.Resource) bool {
				return item.Name == collectionName
			})

			if !exists {
				resources = append(resources, &deploymentspb.Resource{
					Name:   collectionName,
					Type:   resourcespb.ResourceType_Collection,
					Config: &deploymentspb.Resource_Collection{},
				})
			}
		}
	}

	return resources, nil
}

// buildPolicyRequirements gathers, compacts, and deduplicates all policy requirements
// compaction is done by grouping policies by their principals and actions
// i.e. two or more policies with identical principals and actions, but different resources, will be combined into a single policy covering all resources.
func buildPolicyRequirements(allServiceRequirements []*ServiceRequirements, projectErrors *ProjectErrors) ([]*deploymentspb.Resource, error) {
	resources := []*deploymentspb.Resource{}

	for _, serviceRequirements := range allServiceRequirements {
		compactedPoliciesByKey := lo.GroupBy(lo.Values(serviceRequirements.policies), func(item *resourcespb.PolicyResource) string {
			// get the princpals and actions as a unique key (make sure they're sorted for consistency)
			principalNames := lo.Reduce(item.Principals, func(agg []string, principal *resourcespb.Resource, idx int) []string {
				return append(agg, principal.Name)
			}, []string{})
			slices.Sort(principalNames)

			principals := strings.Join(principalNames, ":")

			slices.Sort(item.Actions)
			actions := lo.Reduce(item.Actions, func(agg string, action resourcespb.Action, idx int) string {
				return agg + action.String()
			}, "")

			return principals + "-" + actions
		})

		compactedPolicies := []*resourcespb.PolicyResource{}
		// for each key of the compacted policies we want to make a single policy object that appends all of the policies resources together
		for _, pols := range compactedPoliciesByKey {
			newPol := pols[0]

			for _, pol := range pols[1:] {
				newPol.Resources = append(newPol.Resources, pol.Resources...)
			}

			compactedPolicies = append(compactedPolicies, newPol)
		}

		dedupedPolicies := map[string]*resourcespb.PolicyResource{}

		for _, v := range compactedPolicies {
			policyName, err := policyResourceName(v)
			if err != nil {
				return nil, err
			}

			dedupedPolicies[policyName] = v
		}

		for policyName, policy := range dedupedPolicies {
			principals := []*deploymentspb.Resource{}
			policyResources := []*deploymentspb.Resource{}

			for _, r := range policy.Resources {
				policyResources = append(policyResources, &deploymentspb.Resource{
					Name: r.Name,
					Type: r.Type,
				})
			}

			for _, p := range policy.Principals {
				principals = append(principals, &deploymentspb.Resource{
					Name: p.Name,
					Type: p.Type,
				})
			}

			res := &deploymentspb.Resource{
				Name: policyName,
				Type: resourcespb.ResourceType_Policy,
				Config: &deploymentspb.Resource_Policy{
					Policy: &deploymentspb.Policy{
						Principals: principals,
						Actions:    policy.Actions,
						Resources:  policyResources,
					},
				},
			}

			resources = append(resources, res)
		}
	}

	return resources, nil
}

func checkServiceRequirementErrors(allServiceRequirements []*ServiceRequirements) error {
	allServiceErrors := []error{}

	for _, serviceRequirements := range allServiceRequirements {
		serviceRequirementsErrors := serviceRequirements.Error()
		if serviceRequirementsErrors != nil {
			allServiceErrors = append(allServiceErrors, serviceRequirementsErrors)
		}
	}

	if len(allServiceErrors) > 0 {
		return errors.Join(allServiceErrors...)
	}

	return nil
}

// convert service requirements to a cloud bill of materials
func ServiceRequirementsToSpec(projectName string, environmentVariables map[string]string, allServiceRequirements []*ServiceRequirements) (*deploymentspb.Spec, error) {
	if err := checkServiceRequirementErrors(allServiceRequirements); err != nil {
		return nil, err
	}

	projectErrors := &ProjectErrors{}

	newSpec := &deploymentspb.Spec{
		Resources: []*deploymentspb.Resource{},
	}

	bucketResources, err := buildBucketRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, bucketResources...)

	topicResources, err := buildTopicRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, topicResources...)

	secretResrources, err := buildSecretRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, secretResrources...)

	websocketResources, err := buildWebsocketRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, websocketResources...)

	scheduleResources, err := buildScheduleRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, scheduleResources...)

	httpResources, err := buildHttpRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, httpResources...)

	apiResources, err := buildApiRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, apiResources...)

	collectionResources, err := buildCollectionsRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, collectionResources...)

	policyResources, err := buildPolicyRequirements(allServiceRequirements, projectErrors)
	if err != nil {
		return nil, err
	}
	newSpec.Resources = append(newSpec.Resources, policyResources...)

	for _, serviceRequirements := range allServiceRequirements {
		newSpec.Resources = append(newSpec.Resources, &deploymentspb.Resource{
			Name: serviceRequirements.serviceName,
			Type: resourcespb.ResourceType_Function,
			Config: &deploymentspb.Resource_ExecutionUnit{
				ExecutionUnit: &deploymentspb.ExecutionUnit{
					Source: &deploymentspb.ExecutionUnit_Image{
						Image: &deploymentspb.ImageSource{
							Uri: fmt.Sprintf(serviceRequirements.serviceName),
						},
					},
					Workers: int32(serviceRequirements.WorkerCount()),
					Type:    serviceRequirements.serviceType,
					Env:     environmentVariables,
				},
			},
		})
	}

	return newSpec, projectErrors.Error()
}

func ApisToOpenApiSpecs(apiRegistrationRequests map[string][]*apispb.RegistrationRequest, projectErrors *ProjectErrors) ([]*openapi3.T, error) {
	specs := []*openapi3.T{}
	apiResources := map[string]*resourcespb.ApiResource{}

	// transform apiRegistrationRequests into routes and apis for ServiceRequirements call
	for apiName := range apiRegistrationRequests {
		apiResources[apiName] = &resourcespb.ApiResource{}
	}

	requirements := []*ServiceRequirements{{routes: apiRegistrationRequests, apis: apiResources}}

	apiRequirements, err := buildApiRequirements(requirements, projectErrors)
	if err != nil {
		return nil, err
	}

	// convert back to openapi for dashboard json
	for _, r := range apiRequirements {
		openapiString := r.GetApi().GetOpenapi()

		// Unmarshal the OpenAPI JSON string into openapi3.T
		var openapiDoc *openapi3.T
		err := json.Unmarshal([]byte(openapiString), &openapiDoc)
		if err != nil {
			return nil, err
		}

		specs = append(specs, openapiDoc)
	}

	// sort apis by title
	slices.SortFunc(specs, func(a, b *openapi3.T) int {
		if a.Info.Title < b.Info.Title {
			return -1
		}

		return 1
	})

	return specs, nil
}
