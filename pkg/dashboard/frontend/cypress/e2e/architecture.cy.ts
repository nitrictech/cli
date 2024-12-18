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
  ':',
  'my-db',
  'my-second-db',
  'services/my-test-service.ts',
  'services/my-test-db.ts',
  'services/my-test-secret.ts',
  'my-first-secret',
  'my-second-secret',
]

describe('Architecture Spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/architecture')
  })

  it('should retrieve correct arch nodes', () => {
    cy.wait(500)

    expectedNodes.forEach((content) => {
      cy.log(`Checking that node: ${content} exists`)
      expect(cy.contains('.react-flow__node', content)).to.exist
    })
  })

  it('should have correct routes drawer content', () => {
    const expected = [
      [
        'edge-label-e-first-api-allmethodsget-services/my-test-service.ts',
        'DELETE/all-methodsGET/all-methodsOPTIONS/all-methodsPATCH/all-methodsPOST/all-methodsPUT/all-methodsGET/header-testPOST/json-testGET/path-test/{name}GET/query-testGET/schedule-countGET/topic-count',
      ],
      [
        'edge-label-e-second-api-imagefrombucketget-services/my-test-service.ts',
        'GET/content-type-binaryGET/content-type-cssGET/content-type-htmlGET/content-type-imageGET/content-type-xmlDELETE/image-from-bucketGET/image-from-bucketPUT/image-from-bucketPUT/very-nested-files',
      ],
      [
        'edge-label-e-my-secret-api-setbinarypost-services/my-test-secret.ts',
        'GET/getPOST/setPOST/set-binary',
      ],
      ['edge-label-e-my-db-api-getget-services/my-test-db.ts', 'GET/get'],
    ]

    expected.forEach(([edge, routes]) => {
      cy.getTestEl(edge).click({
        force: true,
      })

      cy.getTestEl('api-routes-list').should('have.text', routes)
    })
  })
})
