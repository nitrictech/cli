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

package ghissue

import (
	"net/url"
	"os"
	"testing"
)

func TestBugLink(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"test"}
	got := BugLink("oops, I did it again!")
	os.Args = origArgs

	gotUrl, err := url.Parse(got)
	if err != nil {
		t.Errorf("BugLink() ulr error = %v", err)
	}

	wantHost := "github.com"
	if gotUrl.Host != wantHost {
		t.Errorf("BugLink() host = %v, want %v", gotUrl.Host, wantHost)
	}

	wantPath := "/nitrictech/cli/issues/new"
	if gotUrl.Path != wantPath {
		t.Errorf("BugLink() path = %v, want %v", gotUrl.Path, wantPath)
	}

	gotQ := gotUrl.Query()

	wantTitle := "Command 'test' panicked: oops, I did it again!"
	if gotQ["title"][0] != wantTitle {
		t.Errorf("BugLink() title = %v, want %v", gotQ["title"][0], wantTitle)
	}
}
