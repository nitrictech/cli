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

package preview

type Feature = string

const (
	Feature_DockerProviders Feature = "docker-providers"
	Feature_BetaProviders   Feature = "beta-providers"
	Feature_SqlDatabases    Feature = "sql-databases"
	Feature_BatchServices   Feature = "batch-services"
	Feature_Websites        Feature = "websites"
)
