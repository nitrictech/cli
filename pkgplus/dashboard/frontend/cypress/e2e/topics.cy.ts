describe('Topics Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/topics')
  })

  it('should retrieve correct topics', () => {
    cy.get('h2').should('contain.text', 'subscribe-tests')
    cy.getTestEl('Topics-count').should('have.text', '2')

    const expectedTopics = ['subscribe-tests', 'subscribe-tests-2']

    expectedTopics.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })
  ;['subscribe-tests', 'subscribe-tests-2'].forEach((topic) => {
    it(`should trigger topic ${topic}`, () => {
      cy.get(`[data-rct-item-id="${topic}"]`).click()

      cy.getTestEl('generated-request-path').should(
        'have.attr',
        'href',
        `http://localhost:4000/topic/${topic}`,
      )

      cy.getTestEl('trigger-topics-btn').click()

      cy.getAPIResponseCodeEditor().should(
        'have.text',
        '1 successful & 0 failed deliveries',
      )
    })
  })

  it(`should add to doc count after topic triggers`, () => {
    cy.visit('/')

    cy.get('[data-rct-item-id="first-api-/topic-count-GET"]').click()

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
