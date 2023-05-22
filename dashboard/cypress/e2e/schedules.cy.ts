describe("Schedules Spec", () => {
  beforeEach(() => {
    cy.viewport("macbook-16");
    cy.visit("/schedules");
  });

  it("should retrieve correct schedules", () => {
    cy.get("h2").should("contain.text", "Schedule - process-tests");
    cy.get(".bg-gray-50 > .bg-white").should("have.text", "2");
    cy.getTestEl("schedules-select").within(() => cy.get("button").click());

    const expectedSchedules = ["process-tests", "process-tests-2"];

    cy.getTestEl("schedules-select-options")
      .find("li")
      .should("have.length", expectedSchedules.length)
      .each(($li, i) => {
        // Assert that each list item contains the expected text
        expect(expectedSchedules).to.include($li.text());
      });
  });

  ["process-tests", "process-tests-2"].forEach((schedule) => {
    it(`should trigger schedule ${schedule}`, () => {
      cy.getTestEl("schedules-select").within(() => cy.get("button").click());

      cy.getTestEl("schedules-select-options").within(() => {
        cy.get("li")
          .contains(new RegExp(`^${schedule}$`))
          .click();
      });

      cy.getTestEl("generated-request-path").should(
        "have.attr",
        "href",
        `http://localhost:4000/topic/${schedule}`
      );

      cy.getTestEl("trigger-schedules-btn").click();

      cy.getAPIResponseCodeEditor().should(
        "have.text",
        "1 successful & 0 failed deliveries"
      );
    });
  });

  it(`should add to doc count after schedule triggers`, () => {
    cy.visit("/");
    cy.getTestEl("endpoint-select").within(() => cy.get("button").click());

    cy.getTestEl("endpoint-select-options").within(() => {
      cy.get("li").contains("first-api/schedule-count1 methods").click();
    });

    cy.getTestEl("send-api-btn").click();

    cy.getAPIResponseCodeEditor()
      .invoke("text")
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({
          firstCount: 1,
          secondCount: 1,
        });
      });
  });
});
