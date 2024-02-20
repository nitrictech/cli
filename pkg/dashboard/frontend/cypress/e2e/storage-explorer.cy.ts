describe('Storage Explorer spec', () => {
  beforeEach(() => {
    cy.viewport('macbook-16')
    cy.visit('/storage')
    cy.wait(500)
  })

  it('should retrieve correct buckets', () => {
    cy.get('h2').first().should('have.text', 'test-bucket')

    const expectedBuckets = ['test-bucket']

    cy.getTestEl('Storage-count').should('have.text', '1')

    expectedBuckets.forEach((id) => {
      cy.get(`[data-rct-item-id="${id}"]`).should('exist')
    })
  })

  it('should load files of first bucket', () => {
    cy.intercept('/api/storage?action=list-files*').as('getFiles')

    cy.wait('@getFiles')

    cy.get('button[title="test-bucket"]').should('exist')
  })

  it('should upload a file to bucket', () => {
    cy.intercept('/api/storage?action=write-file*').as('writeFile')
    cy.fixture('photo.jpg').then((fileContent) => {
      // Use cy.get() to select the file input element and upload the file
      cy.getTestEl('file-upload').then((el) => {
        // Upload the file to the input element
        const testFile = new File([fileContent], 'storage-test-photo.jpg', {
          type: 'image/jpeg',
        })
        const dataTransfer = new DataTransfer()
        dataTransfer.items.add(testFile)
        const fileInput = el[0]
        // @ts-ignore
        fileInput.files = dataTransfer.files
        // Trigger a 'change' event on the input element
        cy.wrap(fileInput).trigger('change', { force: true })
      })
    })

    cy.wait('@writeFile')

    cy.get('[data-chonky-file-id="storage-test-photo.jpg"]', {
      timeout: 5000,
    }).should('exist')
  })

  it('should remove a file from bucket', () => {
    cy.get('[data-chonky-file-id="storage-test-photo.jpg"]').type('{del}')

    cy.get('[data-chonky-file-id="storage-test-photo.jpg"]').should('not.exist')
  })

  it('should handle directories', () => {
    cy.fixture('photo.jpg').then((fileContent) => {
      // Use cy.get() to select the file input element and upload the file
      cy.getTestEl('file-upload').then((el) => {
        // Upload the file to the input element
        const testFile = new File(
          [fileContent],
          '5/4/3/2/1/storage-test-photo.jpg',
          {
            type: 'image/jpeg',
          },
        )
        const dataTransfer = new DataTransfer()
        dataTransfer.items.add(testFile)
        const fileInput = el[0]
        // @ts-ignore
        fileInput.files = dataTransfer.files
        // Trigger a 'change' event on the input element
        cy.wrap(fileInput).trigger('change', { force: true })
      })
    })
    ;['5/', '5/4/', '5/4/3/', '5/4/3/2/', '5/4/3/2/1/'].forEach((id) => {
      cy.get(`[data-chonky-file-id="${id}"]`, {
        timeout: 5000,
      }).dblclick()
    })

    cy.get(`[data-chonky-file-id="5/4/3/2/1/storage-test-photo.jpg"]`, {
      timeout: 5000,
    }).should('exist')
  })

  it('should delete directory files', () => {
    cy.get(`[data-chonky-file-id="5/`, {
      timeout: 5000,
    }).type('{del}')

    cy.get(`[data-chonky-file-id="5/`, {
      timeout: 5000,
    }).should('not.exist')
  })
})
