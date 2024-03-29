Cypress.Commands.add('checkItemOnGenericListLink', resourceName => {
    cy.get('ui5-table-row')
        .find('ui5-table-cell')
        .contains('span', resourceName)
        .should('be.visible');
});

Cypress.Commands.add('deleteFromGenericList', (resourceName) => {
    cy.get('ui5-combobox[placeholder="Search"]')
        .find('input')
        .click()
        .type(resourceName, {force: true});

    cy.get('ui5-button[data-testid="delete"]').click();

});

