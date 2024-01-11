package run

import (
	"path/filepath"

	"github.com/nitrictech/nitric/core/pkg/env"
)

// Base directory used for all temporary files, such as logs, etc.
var NITRIC_TMP = "./.nitric"

// Base directory for temporary files used for local development, e.g. files in buckets, collections, etc.
var NITRIC_LOCAL_RUN_DIR = env.GetEnv("NITRIC_LOCAL_RUN_DIR", filepath.Join(NITRIC_TMP, "./run/"))

// Local run temporary files sub-directories
var LOCAL_DB_DIR = env.GetEnv("LOCAL_DB_DIR", filepath.Join(NITRIC_LOCAL_RUN_DIR.String(), "./collections/"))
var LOCAL_BUCKETS_DIR = env.GetEnv("LOCAL_BUCKETS_DIR", filepath.Join(NITRIC_LOCAL_RUN_DIR.String(), "./buckets/"))
var LOCAL_SEAWEED_LOGS_DIR = env.GetEnv("LOCAL_SEAWEED_LOGS_DIR", filepath.Join(NITRIC_LOCAL_RUN_DIR.String(), "./logs/"))
var LOCAL_SECRETS_DIR = env.GetEnv("LOCAL_SECRETS_DIR", filepath.Join(NITRIC_LOCAL_RUN_DIR.String(), "./secrets/"))

var MAX_WORKERS = env.GetEnv("MAX_WORKERS", "300")
