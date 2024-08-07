describe('Secrets Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/secrets')
    cy.wait(500)
  })

  it('should retrieve correct secrets', () => {
    cy.get('h2').should('contain.text', 'my-first-secret')

    const expectedSecrets = ['my-first-secret', 'my-second-secret']

    expectedSecrets.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })
  ;['my-first-secret', 'my-second-secret'].forEach((sec) => {
    it(`should check for no versions ${sec}`, () => {
      cy.get(`[data-rct-item-id="${sec}"]`).click()

      cy.get('.p-4 > .flex > .text-lg').should(
        'have.text',
        'No versions found.',
      )
    })

    it(`should create latest version for ${sec}`, () => {
      cy.get(`[data-rct-item-id="${sec}"]`).click()

      cy.getTestEl('create-new-version').click()

      cy.intercept({
        url: '/api/secrets?**',
        method: 'POST',
      }).as('secrets')

      cy.getTestEl('secret-value').type(`my-secret-value-${sec}`)

      cy.getTestEl('submit-secrets-dialog').click()

      cy.wait('@secrets')

      cy.wait(500)

      cy.getTestEl('data-table-cell-0_value').should(
        'have.text',
        `my-secret-value-${sec}`,
      )

      cy.getTestEl('data-table-0-latest-badge').should('have.text', 'Latest')
    })

    it(`should create new latest version for ${sec}`, () => {
      cy.get(`[data-rct-item-id="${sec}"]`).click()

      cy.getTestEl('create-new-version').click()

      cy.intercept({
        url: '/api/secrets?**',
        method: 'POST',
      }).as('secrets')

      cy.getTestEl('secret-value').type(`my-secret-value-${sec}-2`)

      cy.getTestEl('submit-secrets-dialog').click()

      cy.wait('@secrets')

      cy.wait(500)

      cy.getTestEl('data-table-cell-0_value').should(
        'have.text',
        `my-secret-value-${sec}-2`,
      )

      cy.getTestEl('data-table-cell-1_value').should(
        'have.text',
        `my-secret-value-${sec}`,
      )

      cy.getTestEl('data-table-0-latest-badge').should('have.text', 'Latest')
    })

    it(`should delete and replace latest ${sec}`, () => {
      cy.get(`[data-rct-item-id="${sec}"]`).click()

      cy.get('[data-testid="data-table-cell-0_select"] > .peer').click({
        force: true,
      })

      cy.getTestEl('delete-selected-versions').click()

      cy.getTestEl('submit-secrets-dialog').click()

      cy.reload()

      cy.get(`[data-rct-item-id="${sec}"]`).click()

      cy.getTestEl('data-table-cell-0_value').should(
        'have.text',
        `my-secret-value-${sec}`,
      )

      cy.getTestEl('data-table-0-latest-badge').should('have.text', 'Latest')
    })
  })

  it(`should retrieve and update value from sdk for my-first-secret`, () => {
    cy.visit('/')

    cy.intercept('/api/call/**').as('apiCall')

    cy.get('[data-rct-item-id="my-secret-api"]').click()

    cy.get('[data-rct-item-id="my-secret-api-/get-GET"]').click()

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(text).to.equal('my-secret-value-my-first-secret')
      })

    cy.get('[data-rct-item-id="my-secret-api-/set-POST"]').click()

    cy.intercept('/api/call/**').as('apiCall')

    cy.getTestEl('Body-tab-btn').click()

    cy.getJSONCodeEditorElement()
      .clear()
      .invoke('html', '{ "my-secret-test": 12345 }')

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.get('[data-rct-item-id="my-secret-api-/get-GET"]').click()

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({
          'my-secret-test': 12345,
        })
      })
  })

  it(`should have latest secret from sdk set call for my-first-secret`, () => {
    cy.get(`[data-rct-item-id="my-first-secret"]`).click()

    cy.getTestEl('data-table-cell-0_value').should(
      'have.text',
      '{"my-secret-test":12345}',
    )

    cy.getTestEl('data-table-0-latest-badge').should('have.text', 'Latest')
  })

  it(`should format Uint8Array secret correctly`, () => {
    cy.visit('/')

    cy.intercept('/api/call/**').as('apiCall')

    cy.get('[data-rct-item-id="my-secret-api"]').click()

    cy.get('[data-rct-item-id="my-secret-api-/set-binary-POST"]').click()

    cy.getTestEl('send-api-btn').click()

    cy.wait('@apiCall')

    cy.visit('/secrets')

    cy.get(`[data-rct-item-id="my-first-secret"]`).click()

    cy.getTestEl('data-table-cell-0_value').should(
      'have.text',
      '00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F \n10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F \n20 21 22 23 24 25 26 27 28 29 2A 2B 2C 2D 2E 2F \n',
    )

    cy.getTestEl('data-table-0-latest-badge').should('have.text', 'Latest')
  })
})
