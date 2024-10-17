describe('APIs spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/')
    cy.wait(500)
  })

  it('should retrieve correct apis and endpoints', () => {
    cy.get('[data-rct-item-id="second-api"]').click()

    const expectedEndpoints = [
      'first-api',
      'first-api-/all-methods-DELETE',
      'first-api-/all-methods-GET',
      'first-api-/all-methods-OPTIONS',
      'first-api-/all-methods-PATCH',
      'first-api-/all-methods-POST',
      'first-api-/all-methods-PUT',
      'first-api-/header-test-GET',
      'first-api-/json-test-POST',
      'first-api-/path-test/{name}-GET',
      'first-api-/query-test-GET',
      'first-api-/schedule-count-GET',
      'first-api-/topic-count-GET',
      'second-api-/content-type-binary-GET',
      'second-api-/content-type-css-GET',
      'second-api-/content-type-html-GET',
      'second-api-/content-type-image-GET',
      'second-api-/content-type-xml-GET',
      'second-api-/image-from-bucket-DELETE',
      'second-api-/image-from-bucket-GET',
      'second-api-/image-from-bucket-PUT',
      'second-api-/very-nested-files-PUT',
    ]

    expectedEndpoints.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })

  it('should allow query params', () => {
    cy.intercept('/api/call/**').as('apiCall')

    cy.get('[data-rct-item-id="first-api-/query-test-GET"]').click()

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.wait(1500)

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({ queryParams: {} })
      })

    cy.getTestEl('query-0-key').type('firstParam')
    cy.getTestEl('query-0-value').type('myValue')

    cy.getTestEl('query-1-key').type('secondParam')
    cy.getTestEl('query-1-value').type('mySecondValue')

    cy.getTestEl('generated-request-path').should(
      'contain.text',
      '/query-test?firstParam=myValue&secondParam=mySecondValue',
    )

    cy.intercept('/api/call/**').as('apiCall')

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.wait(1500)

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({
          queryParams: {
            firstParam: 'myValue',
            secondParam: 'mySecondValue',
          },
        })
      })
  })

  it('should allow request headers', () => {
    cy.intercept('/api/call/**').as('apiCall')

    cy.get('[data-rct-item-id="first-api-/header-test-GET"]').click()

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(JSON.parse(text)).to.have.property('headers')
      })

    cy.intercept('/api/call/**').as('apiCall')

    cy.getTestEl('Headers-tab-btn').first().click()

    cy.getTestEl('header-2-key').clear().type('X-First-Header')
    cy.getTestEl('header-2-value').clear().type('the value')

    cy.getTestEl('header-3-key').clear().type('X-Second-Header')
    cy.getTestEl('header-3-value').clear().type('the second value')

    cy.getTestEl('generated-request-path').should(
      'contain.text',
      '/header-test',
    )

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.wait(1000)

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(JSON.parse(text).headers).to.contain({
          'x-first-header': 'the value',
          'x-second-header': 'the second value',
        })
      })
  })

  it('should allow path params', () => {
    cy.intercept('/api/call/**').as('apiCall')

    cy.get('[data-rct-item-id="first-api-/path-test/{name}-GET"]').click()

    cy.getTestEl('send-api-btn').click()

    cy.getTestEl('path-0-value-error-icon').should('exist')

    cy.getTestEl('path-0-key').should('have.value', 'name')
    cy.getTestEl('path-0-value').type('tester')

    cy.getTestEl('generated-request-path').should(
      'contain.text',
      '/path-test/tester',
    )

    cy.intercept('/api/call/**').as('apiCall')

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.getAPIResponseCodeEditor().should('have.text', 'Hello tester')
  })

  it('should allow json body', () => {
    cy.intercept('/api/call/**').as('apiCall')

    cy.get('[data-rct-item-id="first-api-/json-test-POST"]').click()

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({})
      })

    cy.getTestEl('Body-tab-btn').click()

    cy.getJSONCodeEditorElement()
      .clear()
      .invoke('html', '{ "my-test": 12345, "secondTest": "testing" }')

    cy.getTestEl('generated-request-path').should('contain.text', '/json-test')

    cy.intercept('/api/call/**').as('apiCall')

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.wait(500)

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({
          requestData: {
            'my-test': 12345,
            secondTest: 'testing',
          },
        })
      })
  })

  it('should upload binary file', () => {
    cy.intercept('/api/call/**').as('apiCall')

    cy.get('[data-rct-item-id="second-api"]').click()
    cy.get('[data-rct-item-id="second-api-/image-from-bucket-PUT"]').click()

    cy.getTestEl('Body-tab-btn').click()

    cy.getTestEl('Binary-tab-btn').click()

    cy.fixture('photo.jpg').then((fileContent) => {
      // Use cy.get() to select the file input element and upload the file
      cy.getTestEl('file-upload').then((el) => {
        // Upload the file to the input element
        const testFile = new File([fileContent], 'photo.jpg', {
          type: 'image/jpeg',
        })
        const dataTransfer = new DataTransfer()
        dataTransfer.items.add(testFile)
        const fileInput = el[0]
        // @ts-ignore
        fileInput.files = dataTransfer.files
        // Trigger a 'change' event on the input element
        cy.wrap(fileInput).trigger('change', { force: true })
      })
    })

    cy.getTestEl('file-upload-info').should(
      'contain.text',
      'photo.jpg - 20.05 KB',
    )

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.getTestEl('response-status', 5000).should('contain.text', 'Status: 200')

    cy.wait(500)

    cy.get('[data-rct-item-id="second-api-/image-from-bucket-GET"]').click()

    cy.intercept('/api/call/**').as('apiCall')

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.getTestEl('response-image').should('exist')
  })
  ;[
    [
      'html',
      `<html>      <head>        <title>My Web Page</title>      </head>      <body>        <h1>Welcome to my web page</h1>        <p>This is some sample HTML content.</p>      </body>    </html>`,
    ],
    [
      'css',
      `body {      font-family: Arial, sans-serif;      background-color: #f1f1f1;    }    h1 {      color: blue;    }    p {      color: green;    }`,
    ],
    [
      'xml',
      `<?xml version="1.0" encoding="UTF-8"?>    <data>      <user>        <name>John Doe</name>        <email>john.doe@example.com</email>      </user>      <user>        <name>Jane Smith</name>        <email>jane.smith@example.com</email>      </user>    </data>`,
    ],
    ['image', ``],
    [
      'binary',
      `<?xml version="1.0" encoding="UTF-8"?>    <data>      <user>        <name>John Doe</name>        <email>john.doe@example.com</email>      </user>      <user>        <name>Jane Smith</name>        <email>jane.smith@example.com</email>      </user>    </data>`,
    ],
  ].forEach(([contentType, expected]) => {
    it(`should handle content type ${contentType}`, () => {
      cy.intercept('/api/call/**').as('apiCall')

      cy.get('[data-rct-item-id="second-api"]').click()
      cy.get(
        `[data-rct-item-id="second-api-/content-type-${contentType}-GET"]`,
      ).click()

      cy.getTestEl('send-api-btn').click()

      cy.wait('@apiCall')

      if (contentType === 'binary') {
        cy.getTestEl('response-binary-link').should('exist').click()

        cy.getTestEl('response-binary-link')
          .invoke('attr', 'href')
          .then((href) => {
            cy.log(href || '')
            // Read the downloaded file
            cy.readFile(`cypress/downloads/${href?.split('/')[3]}.xml`).then(
              (fileContent) => {
                expect(fileContent.replace(/[\r\n\t ]/g, '')).to.equal(
                  expected.replace(/[\r\n\t ]/g, ''),
                )
              },
            )
          })
      } else if (contentType === 'image') {
        cy.getTestEl('response-image').should('exist')
      } else {
        cy.getAPIResponseCodeEditor().should('have.text', expected)
      }
    })
  })
})
