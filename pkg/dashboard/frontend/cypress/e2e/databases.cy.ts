const testQueries = `
DROP TABLE IF EXISTS test_table;

CREATE TABLE IF NOT EXISTS test_table (
  id SERIAL PRIMARY KEY,                     -- Primary key column with auto-incrementing ID
  name VARCHAR(100) NOT NULL,                -- Name column with a maximum length of 100 characters
  age INT NOT NULL,                          -- Age column for integer values
  birth_date DATE NOT NULL,                  -- Birth date column for date values
  last_login TIMESTAMP NOT NULL,             -- Last login column for timestamp values
  is_active BOOLEAN NOT NULL,                -- Active status column with boolean values
  balance NUMERIC(10, 2) NOT NULL,           -- Balance column with up to 10 digits, 2 of which are after the decimal point
  profile JSON NOT NULL,                     -- Profile column for JSON data
  preferences JSONB NOT NULL,                -- Preferences column for JSONB data
  small_number SMALLINT NOT NULL,            -- Small number column for small integer values
  big_number BIGINT NOT NULL,                -- Big number column for large integer values
  real_number REAL NOT NULL,                 -- Real number column for single precision floating-point numbers
  double_number DOUBLE PRECISION NOT NULL,  -- Double number column for double precision floating-point numbers
  ip INET NOT NULL,                         -- IP address column
  mac MACADDR NOT NULL,                     -- MAC address column
  bit_field BIT(8) NOT NULL,                -- Bit field column with 8 bits
  char_field CHAR(10) NOT NULL,            -- Character field column with a fixed length of 10 characters
  interval_field INTERVAL NOT NULL,         -- Interval column for time intervals
  bytea_field BYTEA NOT NULL,                -- Byte array column for binary data
  product_id UUID NOT NULL                  -- UUID field for product identification
);

INSERT INTO test_table (
  name, age, birth_date, last_login, is_active, balance, profile, preferences, small_number, 
  big_number, real_number, double_number, ip, mac, bit_field, char_field, interval_field, bytea_field, product_id
) VALUES 
  (
    'Alice', 30, '1993-01-01', '2023-01-01 12:34:56', true, 1234.56, 
    '{"hobbies": ["reading", "swimming"]}', '{"theme": "dark", "notifications": true}', 12, 123456789012345, 3.14, 3.1415926535, 
    '192.168.1.1', '08:00:2b:01:02:03', B'10101010', 'ABCDEFGHIJ', '1 year', '\\xDEADBEEF', 'a01f241c-1a46-4ca6-ab50-b2dcb509b649'
  ),
  (
    'Bob', 45, '1978-02-02', '2023-02-02 14:56:12', false, 7890.12, 
    '{"hobbies": ["cycling", "hiking"]}', '{"theme": "light", "notifications": false}', 34, 987654321098765, 2.71, 2.7182818284, 
    '10.0.0.1', '08:00:2b:04:05:06', B'11110000', 'KLMNOPQRST', '2 months', '\\xC0FFEE', 'f22eae62-6e2f-4539-9858-9108cb9b2014'
  ),
  (
    'Charlie', 28, '1995-03-03', '2023-03-03 16:12:12', true, 4567.89, 
    '{"hobbies": ["gaming", "cooking"]}', '{"theme": "blue", "notifications": true}', 56, 123123123123123, 1.62, 1.6180339887, 
    '172.16.0.1', '08:00:2b:07:08:09', B'00001111', 'UVWXYZABCD', '3 days', '\\xBAADF00D', 'af6d05e9-095f-4b1e-bb12-c3215b383d62'
  ),
  (
    'Diana with special; chars ''', 35, '1988-04-04', '2023-04-04 18:12:12', false, 5678.90, 
    '{"hobbies": ["painting", "running"]}', '{"theme": "red", "notifications": false}', 78, 456456456456456, 0.58, 0.5772156649, 
    '192.168.100.1', '08:00:2b:10:11:12', B'11001100', 'EFGHIJKLMN', '4 hours', '\\xFEEDFACE', '179e1545-ec38-42da-b0cb-8ee20fcb0912'
  );
`

const expectedResults = [
  {
    id: 1,
    name: 'Alice',
    age: 30,
    birth_date: '1993-01-01',
    last_login: '2023-01-01 12:34:56',
    is_active: true,
    balance: 1234.56,
    profile: {
      hobbies: ['reading', 'swimming'],
    },
    preferences: {
      notifications: true,
      theme: 'dark',
    },
    small_number: 12,
    big_number: 123456789012345,
    real_number: 3.14,
    double_number: 3.1415926535,
    ip: '192.168.1.1',
    mac: '08:00:2b:01:02:03',
    bit_field: '10101010',
    char_field: 'ABCDEFGHIJ',
    interval_field: '1 year',
    bytea_field: '\\xdeadbeef',
    product_id: 'a01f241c-1a46-4ca6-ab50-b2dcb509b649',
  },
  {
    id: 2,
    name: 'Bob',
    age: 45,
    birth_date: '1978-02-02',
    last_login: '2023-02-02 14:56:12',
    is_active: false,
    balance: 7890.12,
    profile: {
      hobbies: ['cycling', 'hiking'],
    },
    preferences: {
      notifications: false,
      theme: 'light',
    },
    small_number: 34,
    big_number: 987654321098765,
    real_number: 2.71,
    double_number: 2.7182818284,
    ip: '10.0.0.1',
    mac: '08:00:2b:04:05:06',
    bit_field: '11110000',
    char_field: 'KLMNOPQRST',
    interval_field: '2 mons',
    bytea_field: '\\xc0ffee',
    product_id: 'f22eae62-6e2f-4539-9858-9108cb9b2014',
  },
  {
    id: 3,
    name: 'Charlie',
    age: 28,
    birth_date: '1995-03-03',
    last_login: '2023-03-03 16:12:12',
    is_active: true,
    balance: 4567.89,
    profile: {
      hobbies: ['gaming', 'cooking'],
    },
    preferences: {
      notifications: true,
      theme: 'blue',
    },
    small_number: 56,
    big_number: 123123123123123,
    real_number: 1.62,
    double_number: 1.6180339887,
    ip: '172.16.0.1',
    mac: '08:00:2b:07:08:09',
    bit_field: '00001111',
    char_field: 'UVWXYZABCD',
    interval_field: '3 days',
    bytea_field: '\\xbaadf00d',
    product_id: 'af6d05e9-095f-4b1e-bb12-c3215b383d62',
  },
  {
    id: 4,
    name: "Diana with special; chars '",
    age: 35,
    birth_date: '1988-04-04',
    last_login: '2023-04-04 18:12:12',
    is_active: false,
    balance: 5678.9,
    profile: {
      hobbies: ['painting', 'running'],
    },
    preferences: {
      notifications: false,
      theme: 'red',
    },
    small_number: 78,
    big_number: 456456456456456,
    real_number: 0.58,
    double_number: 0.5772156649,
    ip: '192.168.100.1',
    mac: '08:00:2b:10:11:12',
    bit_field: '11001100',
    char_field: 'EFGHIJKLMN',
    interval_field: '04:00:00',
    bytea_field: '\\xfeedface',
    product_id: '179e1545-ec38-42da-b0cb-8ee20fcb0912',
  },
]

describe('Databases Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/databases')
  })

  it('should retrieve correct databases', () => {
    cy.get('h2').should('contain.text', 'my-db')

    const expectedDbs = ['my-db', 'my-second-db']

    expectedDbs.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })
  ;['my-db', 'my-second-db'].forEach((db, idx) => {
    it(`should check connection string for ${db}`, () => {
      cy.get(`[data-rct-item-id="${db}"]`).click()

      cy.getTestEl('connection-string').should(
        'have.text',
        `postgresql://postgres:localsecret@localhost:5432/${db}?sslmode=disable`,
      )
    })

    it(`should create test table ${db} and see if it exists`, () => {
      cy.get(`[data-rct-item-id="${db}"]`).click()

      cy.get('#sql-editor .cm-content', {
        timeout: 5000,
      })
        .clear({
          force: true,
        })
        .invoke('html', testQueries)

      cy.getTestEl('run-btn').click()

      cy.intercept('POST', '/api/sql', (req) => {
        if (req.body && req.body.query === 'select * from test_table;') {
          req.continue()
        }
      }).as('query')

      cy.get('#sql-editor .cm-content', {
        timeout: 5000,
      })
        .clear({
          force: true,
        })
        .invoke('html', 'select * from test_table;')

      cy.getTestEl('run-btn').click()

      cy.wait('@query').then((interception) => {
        // Validate the response
        expect(interception.response.statusCode).to.equal(200)
        expect(interception.response.body).to.deep.equal(expectedResults)
      })
    })

    it(`should create test table ${db} and see if it exists`, () => {
      cy.get(`[data-rct-item-id="${db}"]`).click()

      cy.get('#sql-editor .cm-content', {
        timeout: 5000,
      })
        .clear({
          force: true,
        })
        .invoke('html', testQueries)

      cy.getTestEl('run-btn').click()

      cy.intercept('POST', '/api/sql', (req) => {
        let body = req.body

        if (typeof req.body === 'string') {
          body = JSON.parse(req.body)
        }

        if (body && body.query.trim() === `select * from test_table;`) {
          req.alias = 'query'
          req.continue()
        } else {
          req.continue()
        }
      })

      cy.get('#sql-editor .cm-content', {
        timeout: 5000,
      })
        .clear({
          force: true,
        })
        .invoke('html', 'select * from test_table;')

      cy.getTestEl('run-btn').click()

      cy.wait('@query').then((interception) => {
        // Validate the response
        expect(interception.response.statusCode).to.equal(200)
        expect(interception.response.body).to.deep.equal(expectedResults)
      })
    })

    it(`should of applied migrations for ${db}`, () => {
      cy.get(`[data-rct-item-id="${db}"]`).click()

      cy.wait(3000)

      cy.intercept('POST', '/api/sql/migrate').as('migrate')

      cy.getTestEl('migrate-btn').click()

      cy.wait('@migrate')

      cy.intercept('POST', '/api/sql', (req) => {
        let body = req.body

        if (typeof req.body === 'string') {
          body = JSON.parse(req.body)
        }

        if (body && body.query.trim() === `select * from my_migration_table;`) {
          req.alias = 'query'
          req.continue()
        } else {
          req.continue()
        }
      })

      cy.get('#sql-editor .cm-content', {
        timeout: 5000,
      })
        .clear({
          force: true,
        })
        .invoke('html', 'select * from my_migration_table;')

      cy.getTestEl('run-btn').click()

      cy.wait('@query').then((interception) => {
        // Validate the response
        expect(interception.response.statusCode).to.equal(200)
        expect(interception.response.body).to.deep.equal([
          {
            id: 1,
            name: `${db}-foo`,
          },
          {
            id: 2,
            name: `${db}-bar`,
          },
        ])
      })
    })
  })
})
