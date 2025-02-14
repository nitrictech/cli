describe('Websites Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
  })

  const expectedWebsites = ['vite-website', 'docs-website']

  it('should retrieve correct websites in list', () => {
    cy.visit('/websites')
    cy.get('h2').should('contain.text', 'vite-website')

    expectedWebsites.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })

  expectedWebsites.forEach((id) => {
    it(`should render website ${id}`, () => {
      cy.visit('/websites')
      cy.get(`[data-rct-item-id="${id}"]`).click()
      cy.get('h2').should('contain.text', id)

      const pathMap = {
        'vite-website': '',
        'docs-website': 'docs',
      }

      const url = `http://localhost:5000/${pathMap[id]}`

      // check iframe url
      cy.get('iframe').should('have.attr', 'src', url)

      cy.visit(url)

      const titleMap = {
        'vite-website': 'Hello Nitric!',
        'docs-website': 'Hello Nitric Docs Test!',
      }

      const title = titleMap[id]

      cy.origin(url, { args: { title } }, ({ title }) => {
        cy.get('h1').should('have.text', title)
      })
    })
  })
})
