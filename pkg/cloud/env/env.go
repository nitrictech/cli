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

package env

import (
	"path/filepath"

	"github.com/nitrictech/nitric/core/pkg/env"
)

// Base directory used for all temporary files, such as logs, etc.
var NITRIC_TMP = "./.nitric"

// Base directory for temporary files used for local development, e.g. files in buckets, key/value stores, etc.
var NITRIC_LOCAL_RUN_DIR = env.GetEnv("NITRIC_LOCAL_RUN_DIR", filepath.Join(NITRIC_TMP, "./run/"))

// Local run temporary files sub-directories
var (
	LOCAL_DB_DIR           = env.GetEnv("LOCAL_DB_DIR", filepath.Join(NITRIC_LOCAL_RUN_DIR.String(), "./kv/"))
	LOCAL_BUCKETS_DIR      = env.GetEnv("LOCAL_BUCKETS_DIR", filepath.Join(NITRIC_LOCAL_RUN_DIR.String(), "./buckets/"))
	LOCAL_SEAWEED_LOGS_DIR = env.GetEnv("LOCAL_SEAWEED_LOGS_DIR", filepath.Join(NITRIC_LOCAL_RUN_DIR.String(), "./logs/"))
	LOCAL_SECRETS_DIR      = env.GetEnv("LOCAL_SECRETS_DIR", filepath.Join(NITRIC_LOCAL_RUN_DIR.String(), "./secrets/"))
)

var MAX_WORKERS = env.GetEnv("MAX_WORKERS", "300")
