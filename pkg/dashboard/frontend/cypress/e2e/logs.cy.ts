describe('logs test suite', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')

    cy.visit('/logs')

    cy.wait(500)
  })

  it(`Should create logs`, () => {
    cy.getTestEl('logs').children().should('have.length.above', 2)

    const expectedMessages = [
      'started service services/my-test-secret.ts',
      'started service services/my-test-service.ts',
      'started service services/my-test-db.ts',
    ]

    expectedMessages.forEach((message) => {
      cy.getTestEl('logs').should('contain.text', message)
    })
  })

  it(`Should search with correct results`, () => {
    cy.getTestEl('logs').children().should('have.length.above', 2)

    cy.getTestEl('log-search').type(
      'started service services/my-test-secret.ts',
    )

    cy.getTestEl('logs').children().should('have.length', 1)

    cy.getTestEl('log-search').clear()

    cy.getTestEl('logs').children().should('have.length.above', 2)
  })

  it(`Should filter origin with correct results`, () => {
    cy.getTestEl('logs').children().should('have.length.above', 2)

    cy.getTestEl('filter-logs-btn').click()

    cy.getTestEl('filter-origin-collapsible').click()

    cy.getTestEl('origin-select').click()

    cy.get('div[data-value="nitric"]').click()

    cy.getTestEl('logs').children().should('have.length.above', 3)

    cy.getTestEl('filter-logs-reset-btn').click()

    cy.getTestEl('logs').children().should('have.length.above', 2)
  })

  it(`Should pre-populate filters via url param`, () => {
    cy.visit(
      '/logs?origin=nitric%2Cservices/my-test-db.ts&level=info&timeline=pastHour',
    )

    cy.getTestEl('filter-logs-btn').click()

    cy.getTestEl('filter-timeline-collapsible').click()
    cy.getTestEl('filter-contains-level-collapsible').click()
    cy.getTestEl('filter-origin-collapsible').click()

    cy.getTestEl('timeline-select-trigger').should('contain.text', 'Past Hour')
    cy.getTestEl('level-select').should('contain.text', 'Info')
    cy.getTestEl('origin-select').should(
      'contain.text',
      'nitricservices/my-test-db.ts',
    )
  })

  it(`Should purge logs`, () => {
    cy.getTestEl('logs').children().should('have.length.above', 2)

    cy.intercept('DELETE', '/api/logs').as('purge')

    cy.getTestEl('log-options-btn').click()

    cy.getTestEl('purge-logs-btn').click()

    cy.wait('@purge')

    cy.getTestEl('logs')
      .children()
      .first()
      .should('have.text', 'No logs available')
  })
})
