package gcp

import (
	"fmt"
	"strings"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/cloudscheduler"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/pubsub"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ScheduleArgs struct {
	Schedule  project.Schedule
	Topics    map[string]*pubsub.Topic
	Functions map[string]*CloudRunner
}

type Schedule struct {
	pulumi.ResourceState

	Name string
	Job  *cloudscheduler.Job
}

// Create a new schedule
func newSchedule(ctx *pulumi.Context, name string, args *ScheduleArgs, opts ...pulumi.ResourceOption) (*Schedule, error) {
	res := &Schedule{
		Name: name,
	}
	normalizedName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	err := ctx.RegisterComponentResource("nitric:schedule:GCPCloudSchedulerJob", name, res, opts...)
	if err != nil {
		return nil, err
	}

	var functionTarget cloudscheduler.JobHttpTargetPtrInput = nil
	switch args.Schedule.Target.Type {
	case "topic":
		// ensure the topic exists and target it
		return nil, fmt.Errorf("schedules targeting topics are deprecated and support has been removed")
	case "function":
		// ensure the function exists and target it
		if f, ok := args.Functions[args.Schedule.Target.Name]; ok {
			functionTarget = cloudscheduler.JobHttpTargetArgs{
				HttpMethod: pulumi.String("POST"),
				Body:       pulumi.String(""),
				Uri:        pulumi.Sprintf("%s/x-nitric-schedule/%s", f.Url, normalizedName),
				OidcToken: cloudscheduler.JobHttpTargetOidcTokenArgs{
					Audience:            f.Url,
					ServiceAccountEmail: f.Invoker.Email,
				},
			}
		}
	default:
		// return an error
		return nil, fmt.Errorf("unsupported schedule trigger type specified")
	}

	if functionTarget == nil {
		return nil, fmt.Errorf("unable to resolve schedule target")
	}

	// If the target type is a function then setup a HttpTarget
	res.Job, err = cloudscheduler.NewJob(ctx, normalizedName, &cloudscheduler.JobArgs{
		TimeZone:   pulumi.String("UTC"),
		HttpTarget: functionTarget,
		Schedule:   pulumi.String(strings.ReplaceAll(args.Schedule.Expression, "'", "")),
	}, pulumi.Parent(res))

	if err != nil {
		return nil, err
	}

	return res, nil
}
