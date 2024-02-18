const expectedNodes = [
  'first-api',
  'second-api',
  'socket',
  'socket-2',
  'socket-3',
  'process-tests',
  'process-tests-2',
  'test-collection',
  'connections',
  'test-bucket',
  'subscribe-tests',
  'subscribe-tests-2',
  'functions/my-test-function.ts',
]

describe('Architecture Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/architecture')
  })

  it('should retrieve correct arch nodes', () => {
    cy.wait(500)

    expectedNodes.forEach((content) => {
      expect(cy.contains('.react-flow__node', content)).to.exist
    })
  })
})
