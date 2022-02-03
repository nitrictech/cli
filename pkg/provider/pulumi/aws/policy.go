package aws

import (
	"encoding/json"
	"fmt"

	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
	iam "github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/s3"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sns"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sqs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Policy struct {
	pulumi.ResourceState

	Name         string
	RolePolicies []*iam.RolePolicy
}

type StackResources struct {
	Topics  map[string]*sns.Topic
	Queues  map[string]*sqs.Queue
	Buckets map[string]*s3.Bucket
}

type PrincipalMap = map[v1.ResourceType]map[string]*iam.Role

type PolicyArgs struct {
	Policy *v1.PolicyResource
	// Resources in the stack that must be protected
	Resources *StackResources
	// Resources in the stack that may act as actors
	Principals PrincipalMap
}

var awsActionsMap map[v1.Action][]string = map[v1.Action][]string{
	v1.Action_BucketFileList: {
		"s3:ListAllMyBuckets",
		"s3:GetBucketTagging",
	},
	v1.Action_BucketFileGet: {
		"s3:GetObject",
	},
	v1.Action_BucketFilePut: {
		"s3:PutObject",
	},
	v1.Action_BucketFileDelete: {
		"s3:DeleteObject",
	},
	v1.Action_TopicList: {
		"sns:ListTopics",
	},
	v1.Action_TopicDetail: {
		"sns:GetTopicAttributes",
	},
	v1.Action_TopicEventPublish: {
		"sns:Publish",
	},
	v1.Action_QueueSend: {
		"sqs:SendMessage",
	},
	v1.Action_QueueReceive: {
		"sqs: ReceiveMessage",
	},
	v1.Action_QueueList: {
		"sqs:ListQueues",
	},
	v1.Action_QueueDetail: {
		"sqs:GetQueueAttributes",
		"sqs:GetQueueUrl",
		"sqs:ListQueueTags",
	},
	v1.Action_CollectionDocumentRead: {
		"dynamodb:GetItem",
		"dynamodb:BatchGetItem",
	},
	v1.Action_CollectionDocumentWrite: {
		"dynamodb:UpdateItem",
	},
	v1.Action_CollectionDocumentDelete: {
		"dynamodb:DeleteItem",
	},
	v1.Action_CollectionQuery: {
		"dynamodb:Query",
		"dynamodb:Scan",
	},
	v1.Action_CollectionList: {
		"dynamodb:ListTables",
	},
}

func actionsToAwsActions(actions []v1.Action) []string {
	awsActions := make([]string, 0)

	for _, a := range actions {
		awsActions = append(awsActions, awsActionsMap[a]...)
	}
	// TODO:
	return awsActions
}

// discover the arn of a deployed resource
func arnForResource(resource *v1.Resource, resources *StackResources) (pulumi.StringOutput, error) {
	switch resource.Type {
	case v1.ResourceType_Bucket:
		if b, ok := resources.Buckets[resource.Name]; ok {
			return b.Arn, nil
		}
	case v1.ResourceType_Topic:
		if t, ok := resources.Topics[resource.Name]; ok {
			return t.Arn, nil
		}
	case v1.ResourceType_Queue:
		if q, ok := resources.Queues[resource.Name]; ok {
			return q.Arn, nil
		}
	default:
		return pulumi.StringOutput{}, fmt.Errorf(
			"invalid resource type: %s. Did you mean to define it as a principal?", resource.Type)
	}

	return pulumi.StringOutput{}, fmt.Errorf("unable to find resource %s::%s", resource.Type, resource.Name)
}

func roleForPrincipal(resource *v1.Resource, principals PrincipalMap) (*iam.Role, error) {
	if pts, ok := principals[resource.Type]; ok {
		if p, ok := pts[resource.Name]; ok {
			return p, nil
		}
	}

	return nil, fmt.Errorf("could not find role for principal: %+v", resource)
}

func newPolicy(ctx *pulumi.Context, name string, args *PolicyArgs, opts ...pulumi.ResourceOption) (*Policy, error) {
	res := &Policy{Name: name, RolePolicies: make([]*iam.RolePolicy, 0)}
	err := ctx.RegisterComponentResource("nitric:func:AWSLambda", name, res, opts...)
	if err != nil {
		return nil, err
	}

	// Get Actions
	actions := actionsToAwsActions(p.Actions)
	// Get principals

	// Get Targets
	targetArns := make([]pulumi.StringOutput, 0, len(p.Resources))
	for _, princ := range args.Policy.Resources {
		if arn, err := arnForResource(princ, args.Resources); err == nil {
			targetArns = append(targetArns, arn)
		} else {
			return nil, err
		}
	}

	// Get principal roles
	// We're collecting roles here to ensure all defined principals are valid before proceeding
	principalRoles := make([]*iam.Role, 0, len(p.Principals))
	for _, princ := range args.Policy.Principals {
		if role, err := roleForPrincipal(princ, args.Principals); err == nil {
			principalRoles = append(principalRoles, role)
		} else {
			return nil, err
		}
	}

	policyJson, err := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Action":   actions,
				"Effect":   "Allow",
				"Resource": targetArns,
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error creating policy document")
	}

	// create role policy for each role
	for _, r := range principalRoles {
		rolePol, err := iam.NewRolePolicy(ctx, name, &iam.RolePolicyArgs{
			Role:   r.ID(),
			Policy: pulumi.String(policyJson),
		}, pulumi.Parent(res))

		if err != nil {
			return nil, err
		}

		res.RolePolicies = append(res.RolePolicies, rolePol)
	}

	return res, nil
}
