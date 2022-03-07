// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-azure/sdk/v4/go/azure/cosmosdb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type MongoCollectionsArgs struct {
	ResourceGroupName pulumi.StringInput
}

type MongoCollections struct {
	pulumi.ResourceState

	Name        string
	Account     *cosmosdb.Account
	MongoDB     *cosmosdb.MongoDatabase
	Collections map[string]*cosmosdb.MongoCollection
}

func (a *azureProvider) newMongoCollections(ctx *pulumi.Context, name string, args *MongoCollectionsArgs, opts ...pulumi.ResourceOption) (*MongoCollections, error) {
	res := &MongoCollections{
		Name:        name,
		Collections: map[string]*cosmosdb.MongoCollection{},
	}
	err := ctx.RegisterComponentResource("nitric:collections:CosmosMongo", name, res, opts...)
	if err != nil {
		return nil, err
	}

	primaryGeo := cosmosdb.AccountGeoLocationArgs{
		FailoverPriority: pulumi.Int(0),
		ZoneRedundant:    pulumi.Bool(false),
		Location:         pulumi.String(a.sc.Region),
	}
	secondaryGeo := cosmosdb.AccountGeoLocationArgs{
		FailoverPriority: pulumi.Int(1),
		ZoneRedundant:    pulumi.Bool(false),
		Location:         pulumi.String("canadacentral"),
	}
	if primaryGeo.Location == secondaryGeo.Location {
		secondaryGeo.Location = pulumi.String("northeurope")
	}

	res.Account, err = cosmosdb.NewAccount(ctx, resourceName(ctx, name, CosmosDBAccountRT), &cosmosdb.AccountArgs{
		ResourceGroupName:  args.ResourceGroupName,
		Kind:               pulumi.String("MongoDB"),
		MongoServerVersion: pulumi.String("4.0"),
		Location:           pulumi.String(a.sc.Region),
		OfferType:          pulumi.String("Standard"),
		ConsistencyPolicy: cosmosdb.AccountConsistencyPolicyArgs{
			ConsistencyLevel: pulumi.String("Eventual"),
		},
		GeoLocations: cosmosdb.AccountGeoLocationArray{primaryGeo, secondaryGeo},
	}, pulumi.Parent(res))
	if err != nil {
		return nil, errors.WithMessage(err, "cosmosdb account")
	}

	res.MongoDB, err = cosmosdb.NewMongoDatabase(ctx, resourceName(ctx, name, MongoDBRT), &cosmosdb.MongoDatabaseArgs{
		ResourceGroupName: args.ResourceGroupName,
		AccountName:       res.Account.Name,
	}, pulumi.Parent(res))
	if err != nil {
		return nil, errors.WithMessage(err, "mongo db")
	}

	for k := range a.proj.Collections {
		res.Collections[k], err = cosmosdb.NewMongoCollection(ctx, resourceName(ctx, k, MongoCollectionRT), &cosmosdb.MongoCollectionArgs{
			ResourceGroupName: args.ResourceGroupName,
			AccountName:       res.Account.Name,
			DatabaseName:      res.MongoDB.Name,
			Indices: cosmosdb.MongoCollectionIndexArray{
				&cosmosdb.MongoCollectionIndexArgs{
					Keys: pulumi.StringArray{
						pulumi.String("_id"),
					},
					Unique: pulumi.Bool(true),
				},
			},
		}, pulumi.Parent(res))
		if err != nil {
			return nil, errors.WithMessage(err, "mongo collection")
		}
	}

	return res, ctx.RegisterResourceOutputs(res, pulumi.Map{
		"name":              pulumi.String(res.Name),
		"mongoDatabaseName": res.MongoDB.Name,
		"connectionString":  res.Account.ConnectionStrings.Index(pulumi.Int(0)),
	})
}
