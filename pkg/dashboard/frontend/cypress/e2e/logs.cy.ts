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
