describe("fathom test suite", () => {
  it(`Should include fathom script`, () => {
    cy.visit("/");
    cy.get("#fathom-script[data-site=FAKE1234]").should("exist");
  });
});
