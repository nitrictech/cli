describe("Topics Spec", () => {
  beforeEach(() => {
    cy.viewport("macbook-16");
    cy.visit("/topics");
  });

  it("should retrieve correct topics", () => {
    cy.get("h2").should("contain.text", "Topic - subscribe-tests");
    cy.get(".bg-gray-50 > .bg-white").should("have.text", "2");
    cy.getTestEl("topics-select").within(() => cy.get("button").click());

    const expectedTopics = ["subscribe-tests", "subscribe-tests-2"];

    cy.getTestEl("topics-select-options")
      .find("li")
      .should("have.length", expectedTopics.length)
      .each(($li, i) => {
        // Assert that each list item contains the expected text
        expect(expectedTopics).to.include($li.text());
      });
  });

  ["subscribe-tests", "subscribe-tests-2"].forEach((topic) => {
    it(`should trigger topic ${topic}`, () => {
      cy.getTestEl("topics-select").within(() => cy.get("button").click());

      cy.getTestEl("topics-select-options").within(() => {
        cy.get("li")
          .contains(new RegExp(`^${topic}$`))
          .click();
      });

      cy.getTestEl("generated-request-path").should(
        "have.attr",
        "href",
        `http://localhost:4000/topic/${topic}`
      );

      cy.getTestEl("trigger-topics-btn").click();

      cy.getAPIResponseCodeEditor().should(
        "have.text",
        "1 successful & 0 failed deliveries"
      );
    });
  });

  it(`should add to doc count after topic triggers`, () => {
    cy.visit("/");
    cy.getTestEl("endpoint-select").within(() => cy.get("button").click());

    cy.getTestEl("endpoint-select-options").within(() => {
      cy.get("li").contains("first-api/topic-count1 methods").click();
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
