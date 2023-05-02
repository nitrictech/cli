describe("Bucket Notifications Spec", () => {
  beforeEach(() => {
    cy.viewport("macbook-16");
    cy.visit("/bucket-notifications");
  });

  const expectedBuckets = [
    {
      bucket: "dog-images",
      notifications: ["Created - /black", "Deleted - /brown"],
    },
    {
      bucket: "cat-images",
      notifications: ["Created - /orange", "Deleted - /spotted"],
    },
  ];

  it("should retrieve correct bucket notifications", () => {
    cy.get("h2").should("contain.text", "Bucket Notification - cat-images");
    cy.get(".bg-gray-50 > .bg-white").should("have.text", "4");
    cy.getTestEl("bucket-select").within(() => cy.get("button").click());
    cy.getTestEl("notification-select").within(() => cy.get("button").click());

    cy.getTestEl("bucket-select-options")
      .find("li")
      .should("have.length", expectedBuckets.length)
      .each(($li, i) => {
        // Assert that each list item contains the expected text
        expect(expectedBuckets.map((b) => b.bucket)).to.include($li.text());
      });

    cy.getTestEl("notification-select-options")
      .find("li")
      .should("have.length", expectedBuckets.length)
      .each(($li, i) => {
        // Assert that each list item contains the expected text
        expect(expectedBuckets.flatMap((b) => b.notifications)).to.include(
          $li.text()
        );
      });
  });

  expectedBuckets.forEach((bucket) => {
    it(`should trigger bucket notification ${bucket.bucket}`, () => {
      cy.getTestEl("bucket-select").within(() => cy.get("button").click());

      cy.getTestEl("bucket-select-options").within(() => {
        cy.get("li")
          .contains(new RegExp(`^${bucket.bucket}$`))
          .click();
      });

      cy.getTestEl("notification-select").within(() =>
        cy.get("button").click()
      );

      cy.getTestEl("notification-select-options").within(() => {
        cy.get("li").contains(bucket.notifications[0]).click();
      });

      cy.getTestEl("generated-request-path").should(
        "have.attr",
        "href",
        `http://localhost:4000/notification/bucket/${bucket.bucket}`
      );

      cy.getTestEl("trigger-notification-btn").click();

      cy.getAPIResponseCodeEditor().should("have.text", "success");
    });
  });
});
