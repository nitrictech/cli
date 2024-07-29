describe('Schedules Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/schedules')
  })

  it('should retrieve correct schedules', () => {
    cy.get('h2').should('contain.text', 'process-tests')

    const expectedSchedules = ['process-tests', 'process-tests-2']

    expectedSchedules.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })
  ;['process-tests', 'process-tests-2'].forEach((schedule) => {
    it(`should trigger schedule ${schedule}`, () => {
      cy.get(`[data-rct-item-id="${schedule}"]`).click()

      cy.getTestEl('generated-request-path').should(
        'have.text',
        `http://localhost:4000/schedules/${schedule}`,
      )

      cy.getTestEl('trigger-schedules-btn').click()

      cy.getAPIResponseCodeEditor().should(
        'have.text',
        'Successfully triggered schedule',
      )
    })
  })

  it(`should add to doc count after schedule triggers`, () => {
    cy.visit('/')

    cy.get('[data-rct-item-id="first-api-/schedule-count-GET"]').click()

    cy.getTestEl('send-api-btn').click()

    cy.getAPIResponseCodeEditor()
      .invoke('text')
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({
          firstCount: 1,
          secondCount: 1,
        })
      })
  })
})
