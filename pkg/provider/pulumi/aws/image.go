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

package aws

import (
	"io/ioutil"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ecr"
	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ECRImageArgs struct {
	LocalImageName  string
	SourceImageName string
	TempDir         string

	AuthToken *ecr.GetAuthorizationTokenResult
}

type ECRImage struct {
	pulumi.ResourceState

	Name        string
	DockerImage *docker.Image
}

func newECRImage(ctx *pulumi.Context, name string, args *ECRImageArgs, opts ...pulumi.ResourceOption) (*ECRImage, error) {
	res := &ECRImage{Name: name}
	err := ctx.RegisterComponentResource("nitric:ECR:Image", name, res, opts...)
	if err != nil {
		return nil, err
	}

	dummyDockerFilePath, err := ioutil.TempFile(args.TempDir, "*.dockerfile")
	if err != nil {
		return nil, err
	}
	_, err = dummyDockerFilePath.WriteString("FROM " + args.SourceImageName + "\n")
	if err != nil {
		return nil, err
	}

	repo, err := ecr.NewRepository(ctx, args.LocalImageName, &ecr.RepositoryArgs{Tags: commonTags(ctx, args.LocalImageName)})
	if err != nil {
		return nil, err
	}

	imageArgs := &docker.ImageArgs{
		ImageName: repo.RepositoryUrl,
		Build: docker.DockerBuildArgs{
			Env: pulumi.StringMap{
				"DOCKER_BUILDKIT": pulumi.String("1"),
			},
			Dockerfile: pulumi.String(dummyDockerFilePath.Name()),
		},
		Registry: docker.ImageRegistryArgs{
			Server:   pulumi.String(args.AuthToken.ProxyEndpoint),
			Username: pulumi.String(args.AuthToken.UserName),
			Password: pulumi.String(args.AuthToken.Password),
		},
	}
	res.DockerImage, err = docker.NewImage(ctx, name+"-image", imageArgs)
	if err != nil {
		return nil, err
	}

	//imageDigest:   this.imageDigest,
	return res, ctx.RegisterResourceOutputs(res, pulumi.Map{
		"name":          pulumi.String(res.Name),
		"imageUri":      res.DockerImage.ImageName,
		"baseImageName": res.DockerImage.BaseImageName,
	})
}
