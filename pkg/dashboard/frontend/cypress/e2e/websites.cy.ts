describe('Websites Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
  })

  const expectedWebsites = ['vite-website', 'docs-website']

  it('should retrieve correct websites in list', () => {
    cy.visit('/websites')
    cy.get('h2').should('contain.text', 'docs-website')

    expectedWebsites.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })

  expectedWebsites.forEach((id) => {
    it(`should render website ${id}`, () => {
      cy.visit('/websites')
      cy.get(`[data-rct-item-id="${id}"]`).click()
      cy.get('h2').should('contain.text', id)

      let originMap = {}

      if (Cypress.env('NITRIC_TEST_TYPE') === 'run') {
        originMap = {
          'vite-website': 'http://localhost:5000',
          'docs-website': 'http://localhost:5000',
        }
      } else {
        originMap = {
          'vite-website': 'http://localhost:5000',
          'docs-website': 'http://localhost:5001',
        }
      }

      const pathMap = {
        'vite-website': '/',
        'docs-website': '/docs',
      }

      // check iframe url
      cy.get('iframe').should('have.attr', 'src', originMap[id] + pathMap[id])

      cy.visit(originMap[id] + pathMap[id])

      const titleMap = {
        'vite-website': 'Hello Nitric!',
        'docs-website': 'Hello Nitric Docs Test!',
      }

      const title = titleMap[id]

      cy.origin(originMap[id], { args: { title } }, ({ title }) => {
        cy.get('h1').should('have.text', title)
      })
    })
  })
})
