describe("API Explorer spec", () => {
  beforeEach(() => {
    cy.viewport("macbook-16");
    cy.visit("/");
  });

  it("should retrieve correct apis and endpoints", () => {
    cy.get("h2").should("contain.text", "API - ");
    cy.get(".bg-gray-50 > .bg-white").should("have.text", "2");
    cy.getTestEl("endpoint-select").within(() => cy.get("button").click());

    const expectedEndpoints = [
      "first-api/all-methods6 methods",
      "first-api/header-test1 methods",
      "first-api/json-test1 methods",
      "first-api/path-test/{name}1 methods",
      "first-api/query-test1 methods",
      "first-api/schedule-count1 methods",
      "second-api/content-type-binary1 methods",
      "second-api/content-type-css1 methods",
      "second-api/content-type-html1 methods",
      "second-api/content-type-image1 methods",
      "second-api/content-type-xml1 methods",
    ];

    cy.getTestEl("endpoint-select-options")
      .find("li")
      .should("have.length", expectedEndpoints.length)
      .each(($li, i) => {
        // Assert that each list item contains the expected text
        expect(expectedEndpoints).to.include($li.text());
      });
  });

  it("should allow query params", () => {
    cy.intercept("/call/**").as("apiCall");
    cy.getTestEl("endpoint-select").within(() => cy.get("button").click());

    cy.getTestEl("endpoint-select-options").within(() => {
      cy.get("li").contains("first-api/query-test1 methods").click();
    });

    cy.getTestEl("send-api-btn").click();

    cy.wait("@apiCall");

    cy.wait(1500);

    cy.getAPIResponseCodeEditor()
      .invoke("text")
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({ queryParams: {} });
      });

    cy.getTestEl("query-0-key").type("firstParam");
    cy.getTestEl("query-0-value").type("myValue");

    cy.getTestEl("query-1-key").type("secondParam");
    cy.getTestEl("query-1-value").type("mySecondValue");

    cy.getTestEl("generated-request-path").should(
      "contain.text",
      "/query-test?firstParam=myValue&secondParam=mySecondValue"
    );

    cy.intercept("/call/**").as("apiCall");

    cy.getTestEl("send-api-btn").click();

    cy.wait("@apiCall");

    cy.wait(1500);

    cy.getAPIResponseCodeEditor()
      .invoke("text")
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({
          queryParams: {
            firstParam: "myValue",
            secondParam: "mySecondValue",
          },
        });
      });
  });

  it("should allow request headers", () => {
    cy.intercept("/call/**").as("apiCall");
    cy.getTestEl("endpoint-select").within(() => cy.get("button").click());

    cy.getTestEl("endpoint-select-options").within(() => {
      cy.get("li").contains("first-api/header-test1 methods").click();
    });

    cy.getTestEl("send-api-btn").click();

    cy.wait("@apiCall");

    cy.getAPIResponseCodeEditor()
      .invoke("text")
      .then((text) => {
        expect(JSON.parse(text)).to.have.property("headers");
      });

    cy.intercept("/call/**").as("apiCall");

    cy.getTestEl("Headers-tab-btn").first().click();

    cy.getTestEl("header-2-key").type("X-First-Header");
    cy.getTestEl("header-2-value").type("the value");

    cy.getTestEl("header-3-key").type("X-Second-Header");
    cy.getTestEl("header-3-value").type("the second value");

    cy.getTestEl("generated-request-path").should(
      "contain.text",
      "/header-test"
    );

    cy.getTestEl("send-api-btn").click();

    cy.wait("@apiCall");

    cy.getAPIResponseCodeEditor()
      .invoke("text")
      .then((text) => {
        expect(JSON.parse(text).headers).to.contain({
          "x-first-header": "the value",
          "x-second-header": "the second value",
        });
      });
  });

  it("should allow path params", () => {
    cy.intercept("/call/**").as("apiCall");
    cy.getTestEl("endpoint-select").within(() => cy.get("button").click());

    cy.getTestEl("endpoint-select-options").within(() => {
      cy.get("li").contains("first-api/path-test/{name}1 methods").click();
    });

    cy.getTestEl("send-api-btn").click();

    cy.wait("@apiCall");

    cy.getAPIResponseCodeEditor().should("contain.text", "Route not found");

    cy.getTestEl("path-0-key").should("have.value", "name");
    cy.getTestEl("path-0-value").type("tester");

    cy.getTestEl("generated-request-path").should(
      "contain.text",
      "/path-test/tester"
    );

    cy.intercept("/call/**").as("apiCall");

    cy.getTestEl("send-api-btn").click();

    cy.wait("@apiCall");

    cy.getAPIResponseCodeEditor().should("have.text", "Hello tester");
  });

  it("should allow json body", () => {
    cy.intercept("/call/**").as("apiCall");
    cy.getTestEl("endpoint-select").within(() => cy.get("button").click());

    cy.getTestEl("endpoint-select-options").within(() => {
      cy.get("li").contains("first-api/json-test1 methods").click();
    });

    cy.getTestEl("send-api-btn").click();

    cy.wait("@apiCall");

    cy.getAPIResponseCodeEditor()
      .invoke("text")
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({ requestData: {} });
      });

    cy.getTestEl("Body-tab-btn").click();

    cy.getJSONCodeEditorElement()
      .clear()
      .invoke("html", '{ "my-test": 12345, "secondTest": "testing" }');

    cy.getTestEl("generated-request-path").should("contain.text", "/json-test");

    cy.intercept("/call/**").as("apiCall");

    cy.getTestEl("send-api-btn").click();

    cy.wait("@apiCall");

    cy.getAPIResponseCodeEditor()
      .invoke("text")
      .then((text) => {
        expect(JSON.parse(text)).to.deep.equal({
          requestData: {
            "my-test": 12345,
            secondTest: "testing",
          },
        });
      });
  });

  [
    [
      "html",
      `<html>      <head>        <title>My Web Page</title>      </head>      <body>        <h1>Welcome to my web page</h1>        <p>This is some sample HTML content.</p>      </body>    </html>`,
    ],
    [
      "css",
      `body {      font-family: Arial, sans-serif;      background-color: #f1f1f1;    }    h1 {      color: blue;    }    p {      color: green;    }`,
    ],
    [
      "xml",
      `<?xml version="1.0" encoding="UTF-8"?>    <data>      <user>        <name>John Doe</name>        <email>john.doe@example.com</email>      </user>      <user>        <name>Jane Smith</name>        <email>jane.smith@example.com</email>      </user>    </data>`,
    ],
    ["image", ``],
    [
      "binary",
      `<?xml version="1.0" encoding="UTF-8"?>    <data>      <user>        <name>John Doe</name>        <email>john.doe@example.com</email>      </user>      <user>        <name>Jane Smith</name>        <email>jane.smith@example.com</email>      </user>    </data>`,
    ],
  ].forEach(([contentType, expected]) => {
    it(`should handle content type ${contentType}`, () => {
      cy.intercept("/call/**").as("apiCall");
      cy.getTestEl("endpoint-select").within(() => cy.get("button").click());

      cy.getTestEl("endpoint-select-options").within(() => {
        cy.get("li")
          .contains(`second-api/content-type-${contentType}1 methods`)
          .click();
      });

      cy.getTestEl("send-api-btn").click();

      cy.wait("@apiCall");

      if (contentType === "binary") {
        cy.getTestEl("response-binary-link").should("exist").click();

        cy.getTestEl("response-binary-link")
          .invoke("attr", "href")
          .then((href) => {
            cy.log(href || "");
            // Read the downloaded file
            cy.readFile(`cypress/downloads/${href?.split("/")[3]}.xml`).then(
              (fileContent) => {
                expect(fileContent.replace(/[\r\n\t ]/g, "")).to.equal(
                  expected.replace(/[\r\n\t ]/g, "")
                );
              }
            );
          });
      } else if (contentType === "image") {
        cy.getTestEl("response-image").should("exist");
      } else {
        cy.getAPIResponseCodeEditor().should("have.text", expected);
      }
    });
  });
});
