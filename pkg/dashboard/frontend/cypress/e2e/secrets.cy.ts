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
      '00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F \n10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F \n20 21 22 23 24 25 26 27 28 29 2A 2B 2C 2D 2E 2F \n30 31 32 33 34 35 36 37 38 39 3A 3B 3C 3D 3E 3F \n40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F \n50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F \n60 61 62 63 64 65 66 67 68 69 6A 6B 6C 6D 6E 6F \n70 71 72 73 74 75 76 77 78 79 7A 7B 7C 7D 7E 7F \n80 81 82 83 84 85 86 87 88 89 8A 8B 8C 8D 8E 8F \n90 91 92 93 94 95 96 97 98 99 9A 9B 9C 9D 9E 9F \nA0 A1 A2 A3 A4 A5 A6 A7 A8 A9 AA AB AC AD AE AF \nB0 B1 B2 B3 B4 B5 B6 B7 B8 B9 BA BB BC BD BE BF \nC0 C1 C2 C3 C4 C5 C6 C7 C8 C9 CA CB CC CD CE CF \nD0 D1 D2 D3 D4 D5 D6 D7 D8 D9 DA DB DC DD DE DF \nE0 E1 E2 E3 E4 E5 E6 E7 E8 E9 EA EB EC ED EE EF \nF0 F1 F2 F3 F4 F5 F6 F7 F8 F9 FA FB FC FD FE FF \n00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F \n10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F \n20 21 22 23 24 25 26 27 28 29 2A 2B 2C 2D 2E 2F \n30 31 32 33 34 35 36 37 38 39 3A 3B 3C 3D 3E 3F \n40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F \n50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F \n60 61 62 63 64 65 66 67 68 69 6A 6B 6C 6D 6E 6F \n70 71 72 73 74 75 76 77 78 79 7A 7B 7C 7D 7E 7F \n80 81 82 83 84 85 86 87 88 89 8A 8B 8C 8D 8E 8F \n90 91 92 93 94 95 96 97 98 99 9A 9B 9C 9D 9E 9F \nA0 A1 A2 A3 A4 A5 A6 A7 A8 A9 AA AB AC AD AE AF \nB0 B1 B2 B3 B4 B5 B6 B7 B8 B9 BA BB BC BD BE BF \nC0 C1 C2 C3 C4 C5 C6 C7 C8 C9 CA CB CC CD CE CF \nD0 D1 D2 D3 D4 D5 D6 D7 D8 D9 DA DB DC DD DE DF \nE0 E1 E2 E3 E4 E5 E6 E7 E8 E9 EA EB EC ED EE EF \nF0 F1 F2 F3 F4 F5 F6 F7 F8 F9 FA FB FC FD FE FF \n00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F \n10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F \n20 21 22 23 24 25 26 27 28 29 2A 2B 2C 2D 2E 2F \n30 31 32 33 34 35 36 37 38 39 3A 3B 3C 3D 3E 3F \n40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F \n50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F \n60 61 62 63 64 65 66 67 68 69 6A 6B 6C 6D 6E 6F \n70 71 72 73 74 75 76 77 78 79 7A 7B 7C 7D 7E 7F \n80 81 82 83 84 85 86 87 88 89 8A 8B 8C 8D 8E 8F \n90 91 92 93 94 95 96 97 98 99 9A 9B 9C 9D 9E 9F \nA0 A1 A2 A3 A4 A5 A6 A7 A8 A9 AA AB AC AD AE AF \nB0 B1 B2 B3 B4 B5 B6 B7 B8 B9 BA BB BC BD BE BF \nC0 C1 C2 C3 C4 C5 C6 C7 C8 C9 CA CB CC CD CE CF \nD0 D1 D2 D3 D4 D5 D6 D7 D8 D9 DA DB DC DD DE DF \nE0 E1 E2 E3 E4 E5 E6 E7 E8 E9 EA EB EC ED EE EF \nF0 F1 F2 F3 F4 F5 F6 F7 F8 F9 FA FB FC FD FE FF \n00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F \n10 11 12 13 14 15 16 17 18 19 1A 1B 1C 1D 1E 1F \n20 21 22 23 24 25 26 27 28 29 2A 2B 2C 2D 2E 2F \n30 31 32 33 34 35 36 37 38 39 3A 3B 3C 3D 3E 3F \n40 41 42 43 44 45 46 47 48 49 4A 4B 4C 4D 4E 4F \n50 51 52 53 54 55 56 57 58 59 5A 5B 5C 5D 5E 5F \n60 61 62 63 64 65 66 67 68 69 6A 6B 6C 6D 6E 6F \n70 71 72 73 74 75 76 77 78 79 7A 7B 7C 7D 7E 7F \n80 81 82 83 84 85 86 87 88 89 8A 8B 8C 8D 8E 8F \n90 91 92 93 94 95 96 97 98 99 9A 9B 9C 9D 9E 9F \nA0 A1 A2 A3 A4 A5 A6 A7 A8 A9 AA AB AC AD AE AF \nB0 B1 B2 B3 B4 B5 B6 B7 B8 B9 BA BB BC BD BE BF \nC0 C1 C2 C3 C4 C5 C6 C7 C8 C9 CA CB CC CD CE CF \nD0 D1 D2 D3 D4 D5 D6 D7 D8 D9 DA DB DC DD DE DF \nE0 E1 E2 E3 E4 E5 E6 E7 E8 E9 EA EB EC ED EE EF \nF0 F1 F2 F3 F4 F5 F6 F7 F8 F9 FA FB FC FD FE FF \n',
    )

    cy.getTestEl('data-table-0-latest-badge').should('have.text', 'Latest')
  })
})
