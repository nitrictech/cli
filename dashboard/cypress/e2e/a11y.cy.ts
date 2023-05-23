describe("a11y test suite", () => {
  const pages = ["/", "/schedules", "/storage", "/not-found"];

  pages.forEach((page) => {
    it(`Should test page ${page} for a11y violations on desktop screen`, () => {
      cy.viewport("macbook-16");
      cy.visit(page);
      cy.wait(1500);
      cy.injectAxe();
      cy.checkA11y(undefined, {
        includedImpacts: ["critical"],
      });
    });

    it(`Should test page ${page} for a11y violations on small screen`, () => {
      cy.viewport("ipad-mini");
      cy.visit(page);
      cy.injectAxe();
      cy.checkA11y(undefined, {
        includedImpacts: ["critical"],
      });
    });
  });
});
