describe('Websockets Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/websockets')
  })

  it('should retrieve correct websockets', () => {
    cy.getTestEl('Websockets-count').should('have.text', '3')

    const expectedWebsockets = ['socket', 'socket-2', 'socket-3']

    expectedWebsockets.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })

  it('should start with 0 connections', () => {
    cy.getTestEl('connections-status').should('have.text', 'Connections: 0')
  })

  it('should have correct websocket url', () => {
    cy.getTestEl('generated-request-path', 5000).should(
      'contain.text',
      'ws://localhost:400',
    )
  })

  it('should go to send messages tab, connect to socket and send a message', () => {
    cy.getTestEl('send-messages-tab-trigger', 5000).click()

    cy.getTestEl('connect-btn').click()

    cy.getTestEl('connected-status').should('have.text', 'Connected')

    cy.getTestEl('accordion-message-0').should(
      'contain.text',
      'Connected to ws://localhost:400',
    )

    cy.getTestEl('message-text-input').type('My awesome test message!')

    cy.getTestEl('send-message-btn').click()

    cy.getTestEl('accordion-message-0').should(
      'have.text',
      'My awesome test message!',
    )
  })

  it('should record message in monitor tab', () => {
    cy.getTestEl('accordion-message-0', 5000).should(
      'have.text',
      'My awesome test message!',
    )
  })

  it('should update connections number', () => {
    cy.getTestEl('send-messages-tab-trigger', 5000).click()

    cy.getTestEl('connect-btn').click()

    cy.getTestEl('monitor-tab-trigger', 5000).click()

    cy.getTestEl('connections-status').should('have.text', 'Connections: 1')
  })

  it('should clear messages in monitor', () => {
    cy.getTestEl('clear-messages-btn', 5000).click()

    cy.getTestEl('accordion-message-0').should('not.exist')
  })

  it('should handle errors in the connect callback', () => {
    cy.get(`[data-rct-item-id="socket-3"]`).click()

    cy.getTestEl('send-messages-tab-trigger', 5000).click()

    cy.getTestEl('connect-btn').click()

    cy.getTestEl('connected-status').should('have.text', 'Disconnected')

    cy.getTestEl('accordion-message-0').should(
      'contain.text',
      'Disconnected from ws://localhost:400',
    )

    cy.getTestEl('accordion-message-1').should(
      'contain.text',
      'Error connecting to ws://localhost:400',
    )
  })

  it('should handle query params', () => {
    cy.getTestEl('send-messages-tab-trigger', 5000).click()

    cy.getTestEl('query-0-key').type('firstParam')
    cy.getTestEl('query-0-value').type('myValue')

    cy.getTestEl('query-1-key').type('secondParam')
    cy.getTestEl('query-1-value').type('mySecondValue')

    cy.getTestEl('generated-request-path').should(
      'contain.text',
      '?firstParam=myValue&secondParam=mySecondValue',
    )

    cy.getTestEl('connect-btn').click()

    cy.getTestEl('connected-status').should('have.text', 'Connected')

    cy.getTestEl('accordion-message-0').should(
      'contain.text',
      '?firstParam=myValue&secondParam=mySecondValue',
    )

    cy.getTestEl('message-text-input').type('My awesome test message!')

    cy.getTestEl('send-message-btn').click()

    cy.wait(1500)

    cy.getTestEl('accordion-message-0').should(
      'have.text',
      'My awesome test message!',
    )
  })
})
