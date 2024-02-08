/// <reference types="cypress" />
// ***********************************************
// This example commands.ts shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add('login', (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add('drag', { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add('dismiss', { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite('visit', (originalFn, url, options) => { ... })
//
// declare global {
//   namespace Cypress {
//     interface Chainable {
//       login(email: string, password: string): Chainable<void>
//       drag(subject: string, options?: Partial<TypeOptions>): Chainable<Element>
//       dismiss(subject: string, options?: Partial<TypeOptions>): Chainable<Element>
//       visit(originalFn: CommandOriginalFn, url: string, options: Partial<VisitOptions>): Chainable<Element>
//     }
//   }
// }
declare global {
  namespace Cypress {
    interface Chainable {
      getTestEl(id: string, timeout?: number): Chainable<JQuery<HTMLElement>>
      getAPIResponseCodeEditor(): Chainable<JQuery<HTMLElement>>
      getJSONCodeEditorElement(): Chainable<JQuery<HTMLElement>>
    }
  }
}

// cypress/support/commands.js
Cypress.Commands.add('getTestEl', (id: string, timeout = 3000) => {
  return cy.get(`[data-testid='${id}']:visible`, { timeout: timeout })
})

Cypress.Commands.add('getAPIResponseCodeEditor', () => {
  return cy.get('#api-response .cm-content', {
    timeout: 5000,
  })
})

Cypress.Commands.add('getJSONCodeEditorElement', () => {
  return cy.get('#json-editor .cm-content', {
    timeout: 5000,
  })
})

export {}
