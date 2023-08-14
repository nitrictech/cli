describe("Websockets Spec", () => {
  beforeEach(() => {
    cy.viewport("macbook-16");
    cy.visit("/websockets");
  });

  it("should retrieve correct websockets", () => {
    cy.get("h2").should("contain.text", "socket");
    cy.getTestEl("Topics-count").should("have.text", "2");

    const expectedWebsockets = ["socket", "socket-2"];

    expectedWebsockets.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should("exist");
    });
  });

  it("should start with 0 connections", () => {
    cy.getTestEl("connections-status").should("have.text", "Connections: 0");
  });

  it("should have correct websocket url", () => {
    cy.getTestEl("generated-request-path", 5000).should(
      "have.text",
      "ws://localhost:4003"
    );
  });

  it("should go to send messages tab, connect to socket and send a message", () => {
    cy.getTestEl("send-messages-tab-trigger", 5000).click();

    cy.getTestEl("connect-btn").click();

    cy.getTestEl("connected-status").should("have.text", "Connected");

    cy.getTestEl("accordion-message-0").should(
      "have.text",
      "Connected to ws://localhost:4003"
    );

    cy.getTestEl("message-text-input").type("My awesome test message!");

    cy.getTestEl("send-message-btn").click();

    cy.getTestEl("accordion-message-0").should(
      "have.text",
      "My awesome test message!"
    );
  });

  it("should record message in monitor tab", () => {
    cy.getTestEl("accordion-message-0", 5000).should(
      "have.text",
      "My awesome test message!"
    );
  });

  it("should update connections number", () => {
    cy.getTestEl("send-messages-tab-trigger", 5000).click();

    cy.getTestEl("connect-btn").click();

    cy.getTestEl("monitor-tab-trigger", 5000).click();

    cy.getTestEl("connections-status").should("have.text", "Connections: 1");
  });

  it("should clear messages in monitor", () => {
    cy.getTestEl("clear-messages-btn", 5000).click();

    cy.getTestEl("accordion-message-0").should("not.exist");
  });

  it("should handle query params", () => {
    cy.getTestEl("send-messages-tab-trigger", 5000).click();

    cy.getTestEl("query-0-key").type("firstParam");
    cy.getTestEl("query-0-value").type("myValue");

    cy.getTestEl("query-1-key").type("secondParam");
    cy.getTestEl("query-1-value").type("mySecondValue");

    cy.getTestEl("generated-request-path").should(
      "contain.text",
      "ws://localhost:4003?firstParam=myValue&secondParam=mySecondValue"
    );

    cy.getTestEl("connect-btn").click();

    cy.getTestEl("connected-status").should("have.text", "Connected");

    cy.getTestEl("accordion-message-0").should(
      "have.text",
      "Connected to ws://localhost:4003?firstParam=myValue&secondParam=mySecondValue"
    );

    cy.getTestEl("message-text-input").type("My awesome test message!");

    cy.getTestEl("send-message-btn").click();

    cy.wait(1500);

    cy.getTestEl("accordion-message-0").should(
      "have.text",
      "My awesome test message!"
    );
  });

  //   it(`should trigger topic ${topic}`, () => {
  //     cy.get(`[data-rct-item-id="${topic}"]`).click();

  //     cy.getTestEl("generated-request-path").should(
  //       "have.attr",
  //       "href",
  //       `http://localhost:4000/topic/${topic}`
  //     );

  //     cy.getTestEl("trigger-topics-btn").click();

  //     cy.getAPIResponseCodeEditor().should(
  //       "have.text",
  //       "1 successful & 0 failed deliveries"
  //     );
  //   });
  // });
});
