describe('Topics Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/topics')
  })

  it('should retrieve correct topics', () => {
    cy.get('h2').should('contain.text', 'subscribe-tests')

    const expectedTopics = ['subscribe-tests', 'subscribe-tests-2']

    expectedTopics.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })
  ;['subscribe-tests', 'subscribe-tests-2'].forEach((topic) => {
    it(`should trigger topic ${topic}`, () => {
      cy.get(`[data-rct-item-id="${topic}"]`).click()

      cy.getTestEl('generated-request-path').should(
        'have.text',
        `http://localhost:4002/topics/${topic}`,
      )

      cy.getTestEl('trigger-topics-btn').click()

      cy.getAPIResponseCodeEditor().should(
        'have.text',
        'Successfully delivered message to topic',
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
