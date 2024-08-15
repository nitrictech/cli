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

// Code adapted from https://github.com/jackc/tern/blob/master/migrate/internal/sqlsplit/sqlsplit_test.go
package sql

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSplit(t *testing.T) {
	for i, tt := range []struct {
		sql      string
		expected []string
	}{
		{
			sql:      ``,
			expected: []string{``},
		},
		{
			sql:      `select 42`,
			expected: []string{`select 42`},
		},
		{
			sql:      `select $1`,
			expected: []string{`select $1`},
		},
		{
			sql:      `select 42; select 7;`,
			expected: []string{`select 42;`, `select 7;`},
		},
		{
			sql:      `select 42; select 7`,
			expected: []string{`select 42;`, `select 7`},
		},
		{
			sql:      `select 42, ';' ";"; select 7;`,
			expected: []string{`select 42, ';' ";";`, `select 7;`},
		},
		{
			sql: `select * -- foo ; bar
from users -- ; single line comments
where id = $1;
select 1;
`,
			expected: []string{
				`select * -- foo ; bar
from users -- ; single line comments
where id = $1;`,
				`select 1;`,
			},
		},
		{
			sql: `select * /* ; multi line ;
;
*/
/* /* ; with nesting ; */ */
from users
where id = $1;
select 1;
`,
			expected: []string{
				`select * /* ; multi line ;
;
*/
/* /* ; with nesting ; */ */
from users
where id = $1;`,
				`select 1;`,
			},
		},
		{
			sql: `select 1;

DO $$DECLARE r record;
BEGIN
   FOR r IN SELECT table_schema, table_name FROM information_schema.tables
			WHERE table_type = 'VIEW' AND table_schema = 'public'
   LOOP
	   EXECUTE 'GRANT ALL ON ' || quote_ident(r.table_schema) || '.' || quote_ident(r.table_name) || ' TO webuser';
   END LOOP;
END$$;

select 2;
`,
			expected: []string{
				`select 1;`,
				`DO $$DECLARE r record;
BEGIN
   FOR r IN SELECT table_schema, table_name FROM information_schema.tables
			WHERE table_type = 'VIEW' AND table_schema = 'public'
   LOOP
	   EXECUTE 'GRANT ALL ON ' || quote_ident(r.table_schema) || '.' || quote_ident(r.table_name) || ' TO webuser';
   END LOOP;
END$$;`,
				`select 2;`,
			},
		},
		{
			sql: `select 1;
SELECT $_testął123$hello; world$_testął123$;
select 2;`,
			expected: []string{
				`select 1;`,
				`SELECT $_testął123$hello; world$_testął123$;`,
				`select 2;`,
			},
		},
		{
			sql: `select 1;
SELECT $test`,
			expected: []string{
				`select 1;`,
				`SELECT $test`,
			},
		},
		{
			sql: `select 1;
SELECT $test$ending`,
			expected: []string{
				`select 1;`,
				`SELECT $test$ending`,
			},
		},
		{
			sql: `select 1;
SELECT $test$ending$test`,
			expected: []string{
				`select 1;`,
				`SELECT $test$ending$test`,
			},
		},
		{
			sql: `select 1;
SELECT $test$ (select $$nested$$ || $testing$strings;$testing$) $test$;
select 2;`,
			expected: []string{
				`select 1;`,
				`SELECT $test$ (select $$nested$$ || $testing$strings;$testing$) $test$;`,
				`select 2;`,
			},
		},
	} {
		t.Run(fmt.Sprintf("test SQLSplit: %d", i), func(t *testing.T) {
			actual := SQLSplit(tt.sql)

			if !cmp.Equal(tt.expected, actual) {
				t.Error(cmp.Diff(tt.expected, actual))
			}
		})
	}
}
