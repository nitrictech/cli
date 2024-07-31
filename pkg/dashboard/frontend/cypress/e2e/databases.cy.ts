describe('Databases Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/databases')
  })

  it('should retrieve correct databases', () => {
    cy.get('h2').should('contain.text', 'my-db')

    const expectedDbs = ['my-db', 'my-second-db']

    expectedDbs.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })
  ;['my-db', 'my-second-db'].forEach((db) => {
    it(`should check connection string for ${db}`, () => {
      cy.get(`[data-rct-item-id="${db}"]`).click()

      cy.getTestEl('connection-string').should(
        'have.text',
        `postgresql://postgres:localsecret@localhost:5432/${db}?sslmode=disable`,
      )
    })

    it(`should create test table ${db} and see if it exists`, () => {
      cy.get(`[data-rct-item-id="${db}"]`).click()

      cy.get('#sql-editor .cm-content', {
        timeout: 5000,
      })
        .clear()
        .invoke(
          'html',
          'create table test_table (id serial primary key, name text);',
        )

      cy.getTestEl('run-btn').click()

      cy.get('#sql-editor .cm-content', {
        timeout: 5000,
      })
        .clear()
        .invoke('html', "insert into test_table (name) values ('John');")

      cy.getTestEl('run-btn').click()

      cy.get('#sql-editor .cm-content', {
        timeout: 5000,
      }).invoke('html', 'select * from test_table;')

      cy.getTestEl('run-btn').click()

      cy.get('.rdg-row-even > [aria-colindex="2"] > .w-full').should(
        'have.text',
        '"John"',
      )
    })
  })
})
